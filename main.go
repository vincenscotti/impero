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
	. "github.com/vincenscotti/impero/model"
	"github.com/vincenscotti/impero/templates"
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
	RenderHTML(w, r, templates.HelpPage())
}

func GameHome(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	playerincome := 0
	shares := make([]*SharesPerPlayer, 0)

	if err := tx.Table("shares").Select("DISTINCT company_id, count(company_id) as shares").Where("`owner_id` = ?", header.CurrentPlayer.ID).Group("`company_id`").Order("`shares` desc").Find(&shares).Error; err != nil {
		panic(err)
	}

	for _, sh := range shares {
		cmp := &sh.Company

		if err := tx.Where(sh.CompanyID).Find(cmp).Error; err != nil {
			panic(err)
		}

		_, sh.ValuePerShare = GetCompanyFinancials(cmp, tx)
		playerincome += sh.Shares * sh.ValuePerShare
	}

	shareauctions := make([]*ShareAuction, 0)

	if err := tx.Model(&ShareAuction{}).Preload("Share").Order("`expiration`").Find(&shareauctions).Error; err != nil {
		panic(err)
	}

	for _, sa := range shareauctions {
		if err := tx.Where(sa.Share.CompanyID).Find(&sa.Share.Company).Error; err != nil {
			panic(err)
		}

		participations := make([]*ShareAuctionParticipation, 0)
		if err := tx.Where("`share_auction_id` = ? and `player_id` = ?", sa.ID, header.CurrentPlayer.ID).Find(&participations).Error; err != nil {
			panic(err)
		}

		sa.Participations = participations
	}

	incomingtransfers := make([]*TransferProposal, 0)

	if err := tx.Where("`to_id` = ?", header.CurrentPlayer.ID).Preload("From").Find(&incomingtransfers).Error; err != nil {
		panic(err)
	}

	page := &GameHomeData{HeaderData: header,
		SharesInfo: shares, PlayerIncome: playerincome,
		ShareAuctions: shareauctions, IncomingTransfers: incomingtransfers}

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
	tx := GetTx(r)

	players := make([]*Player, 0)

	if err := tx.Find(&players).Error; err != nil {
		panic(err)
	}

	sort.Stable(sortablePlayers(players))

	page := &EndGameData{Players: players}

	RenderHTML(w, r, templates.EndGamePage(page))
}

func updateGameStatus(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tx := GetTx(r)
		now := GetTime(r)
		opt := GetOptions(r)

		lastcheckpoint := opt.LastCheckpoint

		endturn := opt.LastTurnCalculated.Add(time.Duration(opt.TurnDuration) * time.Minute)

		logger.Println("first endturn is ", endturn)

		for lastcheckpoint.Before(now) {
			if now.Before(endturn) {
				endturn = now
			}

			logger.Println("doing everything between ", lastcheckpoint, " and ", endturn)

			shareauctions := make([]*ShareAuction, 0)
			if err := tx.Preload("Share").Preload("HighestOfferPlayer").Where("`expiration` < ?", endturn).Find(&shareauctions).Error; err != nil {
				panic(err)
			}

			for _, sa := range shareauctions {
				sa.Share.OwnerID = sa.HighestOfferPlayerID

				cmp := &Company{}
				if err := tx.Where(sa.Share.CompanyID).First(cmp).Error; err != nil {
					panic(err)
				}

				cmp.ShareCapital += sa.HighestOffer

				if err := tx.Save(&sa.Share).Error; err != nil {
					panic(err)
				}

				if err := tx.Save(cmp).Error; err != nil {
					panic(err)
				}

				participations := make([]*ShareAuctionParticipation, 0)
				tx.Model(&ShareAuctionParticipation{}).Where("`share_auction_id` = ?", sa.ID).Find(&participations)

				// report generation

				for _, participant := range participations {
					subject := "Asta " + cmp.Name
					content := fmt.Sprintf("L'asta a cui hai partecipato per la societa' "+cmp.Name+" e' stata vinta da "+sa.HighestOfferPlayer.Name+" per %d$.", sa.HighestOffer)
					report := &Report{PlayerID: participant.PlayerID, Date: sa.Expiration, Subject: subject, Content: content}
					if err := tx.Create(report).Error; err != nil {
						panic(err)
					}
				}

				if err := tx.Delete(sa).Error; err != nil {
					panic(err)
				}
				if err := tx.Delete(&ShareAuctionParticipation{}, "share_auction_id = ?", sa.ID).Error; err != nil {
					panic(err)
				}
			}

			transferproposals := make([]*TransferProposal, 0)
			if err := tx.Preload("From").Preload("To").Where("`expiration` < ?", endturn).Find(&transferproposals).Error; err != nil {
				panic(err)
			}

			for _, tp := range transferproposals {
				tp.From.Budget += tp.Amount

				if err := tx.Save(&tp.From).Error; err != nil {
					panic(err)
				}

				// report generation

				subject := "Proposta di trasferimento denaro"
				content := fmt.Sprintf("La proposta di trasferimento di %d$ da "+tp.From.Name+" a "+tp.To.Name+" e' scaduta", tp.Amount)
				report := &Report{PlayerID: tp.FromID, Date: tp.Expiration, Subject: subject, Content: content}

				if err := tx.Create(report).Error; err != nil {
					panic(err)
				}

				report.ID = 0
				report.PlayerID = tp.ToID

				if err := tx.Create(report).Error; err != nil {
					panic(err)
				}

				if err := tx.Delete(tp).Error; err != nil {
					panic(err)
				}
			}

			partnerships := make([]*Partnership, 0)
			if err := tx.Preload("From").Preload("To").Where("`proposal_expiration` < ?", endturn).Find(&partnerships).Error; err != nil {
				panic(err)
			}

			for _, p := range partnerships {
				if !p.ProposalExpiration.IsZero() {
					// report generation

					subject := "Proposta di partnership scaduta"
					content := "La proposta di partnership tra " + p.From.Name + " e " + p.To.Name + " e' scaduta"
					report := &Report{PlayerID: p.From.CEOID, Date: p.ProposalExpiration, Subject: subject, Content: content}

					if err := tx.Create(report).Error; err != nil {
						panic(err)
					}

					report.ID = 0
					report.PlayerID = p.To.CEOID

					if err := tx.Create(report).Error; err != nil {
						panic(err)
					}

					if err := tx.Delete(p).Error; err != nil {
						panic(err)
					}
				}
			}

			if endturn.Before(now) {
				logger.Println("end turn on ", endturn)

				cmps := make([]*Company, 0)
				shareholder := &Player{}

				tx.Find(&cmps)

				type Dividend struct {
					Company *Company
					Income  int
				}
				dividendsPerPlayer := make(map[uint][]Dividend)

				if err := tx.Model(&Player{}).Update("last_income", 0).Error; err != nil {
					panic(err)
				}

				for _, cmp := range cmps {
					pureIncome, valuePerShare := GetCompanyFinancials(cmp, tx)

					shareholders := make([]*ShareholdersPerCompany, 0)

					shares := 0

					if err := tx.Table("shares").Select("DISTINCT owner_id, count(owner_id) as shares").Where("`company_id` = ?", cmp.ID).Where("`owner_id` != 0").Group("owner_id").Find(&shareholders).Error; err != nil {
						panic(err)
					}

					for _, sh := range shareholders {
						shares += sh.Shares
					}

					for _, sh := range shareholders {
						shareholder.ID = 0

						if err := tx.Where(sh.OwnerID).Find(shareholder).Error; err != nil {
							panic(err)
						}

						shareholder.Budget += valuePerShare * sh.Shares
						shareholder.LastIncome += valuePerShare * sh.Shares
						shareholder.LastBudget = shareholder.Budget

						if err := tx.Save(shareholder).Error; err != nil {
							panic(err)
						}

						dividendsPerPlayer[sh.OwnerID] = append(dividendsPerPlayer[sh.OwnerID], Dividend{cmp, valuePerShare * sh.Shares})
					}

					cmp.ShareCapital += int(pureIncome)
					cmp.ActionPoints = opt.CompanyActionPoints + shares

					if err := tx.Save(cmp).Error; err != nil {
						panic(err)
					}
				}

				for shid, dividends := range dividendsPerPlayer {
					subject := fmt.Sprintf("Dividendi turno %d", opt.Turn)
					content := fmt.Sprintf("I dividendi per questo turno sono i seguenti.<br>")

					totalincome := 0
					for _, d := range dividends {
						content += fmt.Sprintf(d.Company.Name+" : %d$<br>", d.Income)
						totalincome += d.Income
					}

					content += fmt.Sprintf("Totale: %d$", totalincome)

					report := &Report{PlayerID: shid, Date: endturn, Subject: subject, Content: content}

					if err := tx.Create(report).Error; err != nil {
						panic(err)
					}
				}

				if err := tx.Model(&Player{}).Update("action_points", opt.PlayerActionPoints).Error; err != nil {
					panic(err)
				}

				opt.LastTurnCalculated = endturn
				opt.Turn += 1
			}

			lastcheckpoint = endturn
			endturn = endturn.Add(time.Duration(opt.TurnDuration) * time.Minute)
		}

		opt.LastCheckpoint = now
		if err := tx.Save(opt).Error; err != nil {
			panic(err)
		}

		next.ServeHTTP(w, r)
	})
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

func main() {
	debug := flag.Bool("debug", true, "turn on debug facilities")
	addr := flag.String("addr", ":8080", "address:port to bind to")
	flag.StringVar(&AdminPass, "pass", "admin", "administrator password")
	dbdriver := flag.String("dbdriver", "mysql", "database driver name")
	dbstring := flag.String("dbstring", "root:root@/testdb?parseTime=true&loc=Local", "database connection string")

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
		&ChatMessage{}, &Company{}, &Partnership{}, &Share{}, &Rental{},
		&ShareAuction{}, &ShareAuctionParticipation{},
		&TransferProposal{})

	binder = NewGorillaBinder()

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	router.HandleFunc("/", GlobalMiddleware(Login)).Name("home")
	router.HandleFunc("/signup/", GlobalMiddleware(Signup)).Name("signup")
	router.HandleFunc("/login/", GlobalMiddleware(Login)).Name("login")
	router.HandleFunc("/logout/", GlobalMiddleware(Logout)).Name("logout")
	router.HandleFunc("/help/", GlobalMiddleware(Help)).Name("help")

	router.HandleFunc("/admin/", GlobalMiddleware(Admin)).Name("admin")
	router.HandleFunc("/admin/options/", GlobalMiddleware(UpdateOptions)).Name("admin_options")
	router.HandleFunc("/admin/map/", GlobalMiddleware(GenerateMap)).Name("admin_map")
	router.HandleFunc("/admin/broadcast/", GlobalMiddleware(BroadcastMessage)).Name("admin_broadcast")

	game := router.PathPrefix("/game").Subrouter()

	game.HandleFunc("/", GameMiddleware(GameHome)).Name("gamehome")

	game.HandleFunc("/player/all/", GameMiddleware(Players)).Name("player_all")
	game.HandleFunc("/player/{id}", GameMiddleware(GetPlayer)).Name("player")
	game.HandleFunc("/player/transfer/", GameMiddleware(Transfer)).Name("player_transfer")
	game.HandleFunc("/player/transfer/confirm/", GameMiddleware(ConfirmTransfer)).Name("player_transfer_confirm")

	game.HandleFunc("/chat/", GameMiddleware(GetChat)).Name("chat")
	game.HandleFunc("/chat/post/", GameMiddleware(PostChat)).Name("chat_post")

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
	game.HandleFunc("/company/addshare/", GameMiddleware(AddShare)).Name("company_addshare")
	game.HandleFunc("/company/buy/", GameMiddleware(BuyNode)).Name("company_buy")
	game.HandleFunc("/company/invest/", GameMiddleware(InvestNode)).Name("company_invest")

	game.HandleFunc("/bid/share/", GameMiddleware(BidShare)).Name("bid_share")
	game.HandleFunc("/map/", GameMiddleware(GetMap)).Name("map")
	game.HandleFunc("/map/costs/{x}/{y}", GameMiddleware(GetCosts)).Name("map_costs")

	s := &http.Server{}
	s.Addr = *addr
	s.Handler = router

	go func() {
		stop := make(chan os.Signal, 1)

		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		<-stop

		ctx, cancel := ctx.WithTimeout(ctx.Background(), time.Minute)
		defer cancel()

		fmt.Println("Trying to shutdown for a minute...")

		if err := s.Shutdown(ctx); err != nil {
			fmt.Println(err)
		}
	}()

	logger.Println(s.ListenAndServe())
}
