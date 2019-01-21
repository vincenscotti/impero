package main

import (
	ctx "context"
	"flag"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/vincenscotti/impero/engine"
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
	"github.com/vincenscotti/impero/tgui"
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
)

var db *gorm.DB
var store sessions.Store
var logger *log.Logger
var router *mux.Router

func Help(w http.ResponseWriter, r *http.Request) {
	//RenderHTML(w, r, templates.HelpPage())
	http.Redirect(w, r, "/static/rules.pdf", http.StatusFound)
}

func GameHome(w http.ResponseWriter, r *http.Request) {
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

type sortablePlayers []*Player

func (sp sortablePlayers) Len() int {
	return len([]*Player(sp))
}

func (sp sortablePlayers) Less(i, j int) bool {
	p := []*Player(sp)
	return p[i].VP > p[j].VP
}

func (sp sortablePlayers) Swap(i, j int) {
	p := []*Player(sp)

	p[i], p[j] = p[j], p[i]
}

func EndGamePage(w http.ResponseWriter, r *http.Request) {
	header := context.Get(r, "header").(*HeaderData)
	tx := GetTx(r)

	winners := make([]*Player, 0)

	err, players := tx.GetPlayers()
	if err != nil {
		panic(err)
	}

	if len(players) > 0 {
		sort.Stable(sortablePlayers(players))
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

func ErrorHandler(err error, w http.ResponseWriter, r *http.Request) {
	session, ok := context.Get(r, "session").(*sessions.Session)

	var pID uint

	if ok {
		pID = session.Values["playerID"].(uint)
	}

	req, _ := httputil.DumpRequest(r, true)
	RenderHTML(w, r, templates.ErrorPage(err, string(req), pID, string(debug.Stack())))
}

var AdminPass string

var gameEngine *engine.Engine

func main() {
	debug := flag.Bool("debug", true, "turn on debug facilities")
	addr := flag.String("addr", ":8080", "address:port to bind to")
	flag.StringVar(&AdminPass, "pass", "admin", "administrator password")
	dbdriver := flag.String("dbdriver", "mysql", "database driver name")
	dbstring := flag.String("dbstring", os.Getenv("MYSQL_CNX_STRING"), "database connection string")
	tgtoken := flag.String("tgtoken", os.Getenv("TGUI_TOKEN"), "telegram bot connection token")
	weburl := flag.String("weburl", os.Getenv("WEB_ROOT"), "URL where the web UI is deployed")

	flag.Parse()

	var err error

	db, err = gorm.Open(*dbdriver, *dbstring)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	store = sessions.NewCookieStore([]byte("secretpassword"))

	logger = log.New(os.Stdout, "impero: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	router = mux.NewRouter()

	if *debug {
		db.LogMode(true)

		router.PathPrefix("/debug/").Handler(http.DefaultServeMux)
	}

	db.AutoMigrate(&Options{}, &Node{}, &Player{}, &Message{}, &Report{},
		&ChatMessage{}, &Company{}, &Partnership{}, &Shareholder{}, &Rental{},
		&ShareAuction{}, &ShareAuctionParticipation{},
		&TransferProposal{}, &ShareOffer{})

	opt := &Options{}
	if err := db.First(opt).Error; err == gorm.ErrRecordNotFound {
		// insert sane default options
		opt.CompanyActionPoints = 5
		opt.CompanyPureIncomePercentage = 30
		opt.CostPerYield = 1.5
		opt.EndGame = 14
		opt.InitialShares = 20
		opt.BlackoutProbPerDollar = 0.001
		opt.StabilityLevels = 5
		opt.MaxBlackoutDeltaPerDollar = 0.0004
		opt.GameStart = time.Now()
		opt.LastTurnCalculated = time.Now()
		opt.NewCompanyCost = 5
		opt.PlayerActionPoints = 8
		opt.PlayerBudget = 10000
		opt.TurnDuration = 5
		opt.Turn = 1

		db.Create(opt)
	}

	binder = NewGorillaBinder()

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	router.HandleFunc("/", GlobalMiddleware(Login)).Name("home")
	router.HandleFunc("/signup/", GlobalMiddleware(Signup)).Name("signup")
	router.HandleFunc("/login/", GlobalMiddleware(Login)).Name("login")
	router.HandleFunc("/logout/", GlobalMiddleware(Logout)).Name("logout")
	router.HandleFunc("/help/", GlobalMiddleware(Help)).Name("help")

	router.HandleFunc("/admin/", GlobalMiddleware(Admin)).Name("admin")
	router.HandleFunc("/admin/options/", GlobalMiddleware(UpdateOptions)).Name("admin_options")
	router.HandleFunc("/admin/map/import/", GlobalMiddleware(ImportMap)).Name("admin_map_import")
	router.HandleFunc("/admin/map/", GlobalMiddleware(GenerateMap)).Name("admin_map")
	router.HandleFunc("/admin/broadcast/", GlobalMiddleware(BroadcastMessage)).Name("admin_broadcast")

	game := router.PathPrefix("/game").Subrouter()

	game.HandleFunc("/", GameMiddleware(GameHome)).Name("gamehome")
	game.HandleFunc("/market/", GameMiddleware(Market)).Name("market")

	game.HandleFunc("/player/all/", GameMiddleware(Players)).Name("player_all")
	game.HandleFunc("/player/{id}", GameMiddleware(GetPlayer)).Name("player")
	game.HandleFunc("/player/transfer/", GameMiddleware(Transfer)).Name("player_transfer")
	game.HandleFunc("/player/transfer/confirm/", GameMiddleware(ConfirmTransfer)).Name("player_transfer_confirm")

	game.HandleFunc("/chat/", GameMiddleware(GetChat)).Name("chat")
	game.HandleFunc("/chat/post/", GameMiddleware(PostChat)).Name("chat_post")

	game.HandleFunc("/message/compose/", GameMiddleware(ComposeMessage)).Name("message_compose")
	game.HandleFunc("/message/inbox/", GameMiddleware(MessagesInbox)).Name("message_inbox")
	game.HandleFunc("/message/outbox/", GameMiddleware(MessagesOutbox)).Name("message_outbox")
	game.HandleFunc("/message/{id}", GameMiddleware(GetMessage)).Name("message")
	game.HandleFunc("/message/new/", GameMiddleware(NewMessagePost)).Name("message_new")

	game.HandleFunc("/report/all/", GameMiddleware(ReportsPage)).Name("report_all")
	game.HandleFunc("/report/{id}", GameMiddleware(ReportPage)).Name("report")
	game.HandleFunc("/report/delete/", GameMiddleware(DeleteReports)).Name("report_delete")

	game.HandleFunc("/company/all/", GameMiddleware(Companies)).Name("company_all")
	game.HandleFunc("/company/{id}", GameMiddleware(GetCompany)).Name("company")
	game.HandleFunc("/company/new/", GameMiddleware(NewCompanyPost)).Name("company_new")
	game.HandleFunc("/company/promoteceo/", GameMiddleware(PromoteCEO)).Name("company_promoteceo")
	game.HandleFunc("/company/partnership/proposal/", GameMiddleware(ProposePartnership)).Name("company_partnership_proposal")
	game.HandleFunc("/company/partnership/confirm/", GameMiddleware(ConfirmPartnership)).Name("company_partnership_confirm")
	game.HandleFunc("/company/partnership/delete/", GameMiddleware(DeletePartnership)).Name("company_partnership_delete")
	game.HandleFunc("/company/pureincome/", GameMiddleware(ModifyCompanyPureIncome)).Name("company_pureincome")
	game.HandleFunc("/company/emitshares/", GameMiddleware(EmitShares)).Name("company_emitshares")
	game.HandleFunc("/company/sellshares/", GameMiddleware(SellShares)).Name("company_sellshares")
	game.HandleFunc("/company/buy/", GameMiddleware(BuyNode)).Name("company_buy")
	game.HandleFunc("/company/invest/", GameMiddleware(InvestNode)).Name("company_invest")

	game.HandleFunc("/stats/", GameMiddleware(Stats)).Name("stats")

	game.HandleFunc("/share/bid/", GameMiddleware(BidShare)).Name("bid_share")
	game.HandleFunc("/share/buy/", GameMiddleware(BuyShare)).Name("buy_share")
	game.HandleFunc("/map/", GameMiddleware(GetMap)).Name("map")

	game.HandleFunc("/chart/", GameMiddleware(EndGamePage)).Name("chart")

	gameEngine = engine.NewEngine(db, logger)

	s := &http.Server{}
	s.Addr = *addr
	s.Handler = router

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
