package main

import (
	ctx "context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"sort"
	"syscall"
	"time"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/vincenscotti/impero/engine"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"github.com/vincenscotti/impero/tgui"
)

type defaultTimeProvider struct{}

func (tp defaultTimeProvider) Now() time.Time {
	return time.Now()
}

func (s *httpBackend) Help(w http.ResponseWriter, r *http.Request) {
	//RenderHTML(w, r, templates.HelpPage())
	http.Redirect(w, r, "/static/rules.pdf", http.StatusFound)
}

func (s *httpBackend) GameHome(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	// TODO: handle errors
	_, shares := tx.GetSharesForPlayer(header.CurrentPlayer)
	_, shpp, playerincome := tx.CalculateSharesIncome(shares)
	_, incomingtransfers := tx.GetIncomingTransfers(header.CurrentPlayer)

	page := &GameHomeData{HeaderData: header,
		SharesInfo: shpp, PlayerIncome: playerincome,
		IncomingTransfers: incomingtransfers}

	RenderHTML(w, r, templates.GameHomePage(page))
}

func (s *httpBackend) EndGamePage(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	winners := make([]*Player, 0)

	err, players := tx.GetPlayers()
	if err != nil {
		panic(err)
	}

	if len(players) > 0 {
		sort.Stable(PlayersSortableByVP(players))
		max := players[0].VP

		for _, p := range players {
			if p.VP == max {
				winners = append(winners, p)
			}
		}
	}

	page := &EndGameData{HeaderData: header, Players: players, Winners: winners}

	RenderHTML(w, r, templates.EndGamePage(page))
}

func (s *httpBackend) ErrorHandler(err error, w http.ResponseWriter, r *http.Request) {
	session, ok := context.Get(r, "session").(*sessions.Session)

	var pID uint

	if ok {
		pID = session.Values["playerID"].(uint)
	}

	req, _ := httputil.DumpRequest(r, true)
	RenderHTML(w, r, templates.ErrorPage(err, string(req), pID, string(debug.Stack())))
}

type httpBackend struct {
	binder     *gorillaBinder
	store      sessions.Store
	logger     *log.Logger
	router     *mux.Router
	AdminPass  string
	gameEngine *engine.Engine
}

func (s httpBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func newHttpBackend(eng *engine.Engine, logger *log.Logger, adminPass string, debug bool) (s httpBackend) {
	s.gameEngine = eng
	s.logger = logger

	s.store = sessions.NewCookieStore([]byte("secretpassword"))

	s.router = mux.NewRouter()

	s.binder = NewGorillaBinder()

	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	s.router.HandleFunc("/api/login/", s.APIMiddleware(s.APILogin)).Name("api_login")

	s.router.HandleFunc("/", s.GlobalMiddleware(s.Login)).Name("home")
	s.router.HandleFunc("/signup/", s.GlobalMiddleware(s.Signup)).Name("signup")
	s.router.HandleFunc("/login/", s.GlobalMiddleware(s.Login)).Name("login")
	s.router.HandleFunc("/logout/", s.GlobalMiddleware(s.Logout)).Name("logout")
	s.router.HandleFunc("/help/", s.GlobalMiddleware(s.Help)).Name("help")

	s.router.HandleFunc("/admin/", s.GlobalMiddleware(s.Admin)).Name("admin")
	s.router.HandleFunc("/admin/options/", s.GlobalMiddleware(s.UpdateOptions)).Name("admin_options")
	s.router.HandleFunc("/admin/map/import/", s.GlobalMiddleware(s.ImportMap)).Name("admin_map_import")
	s.router.HandleFunc("/admin/map/", s.GlobalMiddleware(s.GenerateMap)).Name("admin_map")
	s.router.HandleFunc("/admin/broadcast/", s.GlobalMiddleware(s.BroadcastMessage)).Name("admin_broadcast")

	game := s.router.PathPrefix("/game").Subrouter()

	game.HandleFunc("/", s.GameMiddleware(s.GameHome)).Name("gamehome")
	game.HandleFunc("/market/", s.GameMiddleware(s.Market)).Name("market")

	game.HandleFunc("/player/all/", s.GameMiddleware(s.Players)).Name("player_all")
	game.HandleFunc("/player/{id}", s.GameMiddleware(s.GetPlayer)).Name("player")
	game.HandleFunc("/player/transfer/", s.GameMiddleware(s.Transfer)).Name("player_transfer")
	game.HandleFunc("/player/transfer/confirm/", s.GameMiddleware(s.ConfirmTransfer)).Name("player_transfer_confirm")

	game.HandleFunc("/chat/", s.GameMiddleware(s.GetChat)).Name("chat")
	game.HandleFunc("/chat/post/", s.GameMiddleware(s.PostChat)).Name("chat_post")

	game.HandleFunc("/message/compose/", s.GameMiddleware(s.ComposeMessage)).Name("message_compose")
	game.HandleFunc("/message/inbox/", s.GameMiddleware(s.MessagesInbox)).Name("message_inbox")
	game.HandleFunc("/message/outbox/", s.GameMiddleware(s.MessagesOutbox)).Name("message_outbox")
	game.HandleFunc("/message/{id}", s.GameMiddleware(s.GetMessage)).Name("message")
	game.HandleFunc("/message/new/", s.GameMiddleware(s.NewMessagePost)).Name("message_new")

	game.HandleFunc("/report/all/", s.GameMiddleware(s.ReportsPage)).Name("report_all")
	game.HandleFunc("/report/{id}", s.GameMiddleware(s.ReportPage)).Name("report")
	game.HandleFunc("/report/delete/", s.GameMiddleware(s.DeleteReports)).Name("report_delete")

	game.HandleFunc("/company/all/", s.GameMiddleware(s.Companies)).Name("company_all")
	game.HandleFunc("/company/{id}", s.GameMiddleware(s.GetCompany)).Name("company")
	game.HandleFunc("/company/new/", s.GameMiddleware(s.NewCompanyPost)).Name("company_new")
	game.HandleFunc("/company/promoteceo/", s.GameMiddleware(s.PromoteCEO)).Name("company_promoteceo")
	game.HandleFunc("/company/partnership/proposal/", s.GameMiddleware(s.ProposePartnership)).Name("company_partnership_proposal")
	game.HandleFunc("/company/partnership/confirm/", s.GameMiddleware(s.ConfirmPartnership)).Name("company_partnership_confirm")
	game.HandleFunc("/company/partnership/delete/", s.GameMiddleware(s.DeletePartnership)).Name("company_partnership_delete")
	game.HandleFunc("/company/pureincome/", s.GameMiddleware(s.ModifyCompanyPureIncome)).Name("company_pureincome")
	game.HandleFunc("/company/emitshares/", s.GameMiddleware(s.EmitShares)).Name("company_emitshares")
	game.HandleFunc("/company/sellshares/", s.GameMiddleware(s.SellShares)).Name("company_sellshares")
	game.HandleFunc("/company/buy/", s.GameMiddleware(s.BuyNode)).Name("company_buy")
	game.HandleFunc("/company/invest/", s.GameMiddleware(s.InvestNode)).Name("company_invest")

	game.HandleFunc("/stats/", s.GameMiddleware(s.Stats)).Name("stats")

	game.HandleFunc("/share/bid/", s.GameMiddleware(s.BidShare)).Name("bid_share")
	game.HandleFunc("/share/buy/", s.GameMiddleware(s.BuyShare)).Name("buy_share")
	game.HandleFunc("/map/", s.GameMiddleware(s.GetMap)).Name("map")

	game.HandleFunc("/chart/", s.GameMiddleware(s.EndGamePage)).Name("chart")

	if debug {
		s.router.PathPrefix("/debug/").Handler(http.DefaultServeMux)
	}

	return
}

func main() {
	debug := flag.Bool("debug", true, "turn on debug facilities")
	addr := flag.String("addr", ":8080", "address:port to bind to")
	adminPass := flag.String("pass", "admin", "administrator password")
	dbdriver := flag.String("dbdriver", "mysql", "database driver name")
	dbstring := flag.String("dbstring", os.Getenv("MYSQL_CNX_STRING"), "database connection string")
	tgtoken := flag.String("tgtoken", os.Getenv("TGUI_TOKEN"), "telegram bot connection token")
	weburl := flag.String("weburl", os.Getenv("WEB_ROOT"), "URL where the web UI is deployed")
	jwtPass := flag.String("jwtpass", "ImperoRocks", "password used to sign JWT authentication tokens")

	flag.Parse()

	db, err := gorm.Open(*dbdriver, *dbstring)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	logger := log.New(os.Stdout, "impero: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	gameEngine := engine.NewEngine(db, logger, defaultTimeProvider{}, []byte(*jwtPass))

	if *debug {
		db.LogMode(true)
	}

	httpBackend := newHttpBackend(gameEngine, logger, *adminPass, *debug)

	s := &http.Server{}
	s.Addr = *addr
	s.Handler = httpBackend

	tgui := tgui.New(gameEngine, *tgtoken, *weburl)

	gameEngine.RegisterNotificator(tgui)
	gameEngine.Boot()

	go func() {
		fmt.Println(s.ListenAndServe())
	}()

	go func() {
		fmt.Println(tgui.Run(*debug))
	}()

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	timeoutctx, cancel := ctx.WithTimeout(ctx.Background(), time.Minute)
	defer cancel()

	fmt.Println("Trying to shutdown for a minute...")

	if err := s.Shutdown(timeoutctx); err != nil {
		fmt.Println(err)
	}

	if err := tgui.Shutdown(timeoutctx); err != nil {
		fmt.Println(err)
	}
}
