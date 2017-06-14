package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	. "impero/model"
	"impero/templates"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"time"
)

var db *gorm.DB
var store sessions.Store
var logger *log.Logger
var router *mux.Router

type gorillaBinder struct {
	decoder *schema.Decoder
}

func NewGorillaBinder() *gorillaBinder {
	formDecoder := schema.NewDecoder()
	formDecoder.IgnoreUnknownKeys(true)

	return &gorillaBinder{formDecoder}
}

func (this *gorillaBinder) Bind(i interface{}, r *http.Request) error {
	err := r.ParseForm()

	if err != nil {
		return err
	}

	return this.decoder.Decode(i, r.PostForm)
}

var binder *gorillaBinder

func renderHTML(w http.ResponseWriter, code int, s string) (err error) {
	w.Header().Set("Content-type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, err = w.Write([]byte(s))
	return
}

func Help(w http.ResponseWriter, r *http.Request) {
	renderHTML(w, 200, templates.HelpPage())
}

func GameHome(w http.ResponseWriter, r *http.Request) {
	tx := GetTx(r)
	header := context.Get(r, "header").(*HeaderData)

	shares := make([]*SharesPerPlayer, 0)

	tx.Table("shares").Select("DISTINCT company_id, count(company_id) as shares").Where("owner_id = ?", header.CurrentPlayer.ID).Group("company_id").Order("shares desc").Find(&shares)

	for _, sh := range shares {
		tx.Where(sh.CompanyID).Find(&sh.Company)
	}

	shareauctions := make([]*ShareAuction, 0)
	tx.Model(&ShareAuction{}).Preload("Share").Find(&shareauctions)

	for _, sa := range shareauctions {
		tx.Where(sa.Share.CompanyID).Find(&sa.Share.Company)
		participations := make([]*ShareAuctionParticipation, 0)
		tx.Where("share_auction_id = ? and player_id = ?", sa.ID, header.CurrentPlayer.ID).Find(&participations)
		sa.Participations = participations
	}

	incomingtransfers := make([]*TransferProposal, 0)
	tx.Where("to_id = ?", header.CurrentPlayer.ID).Preload("From").Find(&incomingtransfers)

	page := &GameHomeData{HeaderData: header, SharesInfo: shares,
		ShareAuctions: shareauctions, IncomingTransfers: incomingtransfers}

	renderHTML(w, 200, templates.GameHomePage(page))
}

type votesPerOwner struct {
	OwnerID uint
	Votes   int
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
			tx.Preload("Share").Preload("HighestOfferPlayer").Where("expiration < ?", endturn).Find(&shareauctions)

			for _, sa := range shareauctions {
				sa.Share.OwnerID = sa.HighestOfferPlayerID

				cmp := &Company{}
				if err := tx.Where(sa.Share.CompanyID).First(cmp); err.Error != nil {
					panic(err.Error)
				}

				cmp.ShareCapital += sa.HighestOffer

				if err := tx.Save(sa.Share); err.Error != nil {
					panic(err.Error)
				}

				if err := tx.Save(cmp); err.Error != nil {
					panic(err.Error)
				}

				participations := make([]*ShareAuctionParticipation, 0)
				tx.Model(&ShareAuctionParticipation{}).Where("share_auction_id = ?", sa.ID).Find(&participations)

				// report generation

				for _, participant := range participations {
					subject := "Asta " + cmp.Name
					content := fmt.Sprintf("L'asta a cui hai partecipato per la societa' "+cmp.Name+" e' stata vinta da "+sa.HighestOfferPlayer.Name+" per %d$.", sa.HighestOffer)
					report := &Report{PlayerID: participant.PlayerID, Date: sa.Expiration, Subject: subject, Content: content}
					if err := tx.Create(report); err.Error != nil {
						panic(err.Error)
					}
				}

				if err := tx.Delete(sa); err.Error != nil {
					panic(err.Error)
				}
				if err := tx.Delete(&ShareAuctionParticipation{}, "share_auction_id = ?", sa.ID); err.Error != nil {
					panic(err.Error)
				}
			}

			if endturn.Before(now) {
				logger.Println("end turn on ", endturn)

				cmps := make([]*Company, 0)
				nodes := make([]*Node, 0)
				rentals := make([]*Rental, 0)
				shareholder := &Player{}

				tx.Find(&cmps)

				type Dividend struct {
					Company *Company
					Income  int
				}
				dividendsPerPlayer := make(map[uint][]Dividend)

				if err := tx.Model(&Player{}).Update("last_income", 0); err.Error != nil {
					panic(err.Error)
				}

				for _, cmp := range cmps {
					// company income

					tx.Where("owner_id = ?", cmp.ID).Find(&nodes)

					income := 0
					for _, n := range nodes {
						income += n.Yield
					}

					tx.Preload("Node").Where("tenant_id = ?", cmp.ID).Find(&rentals)
					for _, r := range rentals {
						income += r.Node.Yield / 2
					}

					shareholders := make([]*ShareholdersPerCompany, 0)
					vote := &ElectionVote{}

					shares := 0
					tx.Table("shares").Select("DISTINCT owner_id, count(owner_id) as shares").Where("company_id = ?", cmp.ID).Where("owner_id != 0").Group("owner_id").Find(&shareholders)

					for _, sh := range shareholders {
						vote.ID = 0
						shares += sh.Shares
						tx.Where("company_id = ? and from_id = ?", cmp.ID, sh.OwnerID).Find(vote)
						sh.VotedFor = vote.ToID
					}

					floatIncome := float64(income)
					pureIncome := math.Floor(floatIncome * 0.3)
					valuepershare := int(math.Ceil((floatIncome - pureIncome) / float64(shares)))

					for _, sh := range shareholders {
						shareholder.ID = 0
						tx.Where(sh.OwnerID).Find(shareholder)

						shareholder.Budget += valuepershare * sh.Shares
						shareholder.LastIncome += valuepershare * sh.Shares
						shareholder.LastBudget = shareholder.Budget

						if err := tx.Save(shareholder); err.Error != nil {
							panic(err.Error)
						}

						dividendsPerPlayer[sh.OwnerID] = append(dividendsPerPlayer[sh.OwnerID], Dividend{cmp, valuepershare * sh.Shares})
					}

					cmp.ShareCapital += int(pureIncome)
					cmp.LastIncome = income

					// company elections

					if cmp.CEOExpiration == opt.Turn {
						if len(shareholders) > 1 {
							logger.Println("elections for ", cmp.Name)

							votesreceived := make(map[uint]int)
							maxvotes := 0
							totalvotes := 0
							for _, sh := range shareholders {
								if sh.VotedFor != 0 {
									logger.Println(sh.OwnerID, "voted for", sh.VotedFor, "with", sh.Shares, "shares")
									votesreceived[sh.VotedFor] += sh.Shares
									totalvotes += sh.Shares

									if votesreceived[sh.VotedFor] > maxvotes {
										maxvotes = votesreceived[sh.VotedFor]
									}
								}
							}

							logger.Println("Votes received: ", votesreceived)
							logger.Println("Max votes: ", maxvotes)

							winners := make([]uint, 0, len(shareholders))

							if maxvotes == 0 {
								winners = append(winners, cmp.CEOID)
							} else {
								for k, v := range votesreceived {
									if v == maxvotes {
										winners = append(winners, k)
									}
								}
							}

							logger.Println("Winners: ", winners)

							cmp.CEOID = winners[rand.Intn(len(winners))]

							logger.Println("Winner: ", cmp.CEOID)

							winner := &Player{}
							tx.Where(cmp.CEOID).Find(winner)

							// report generation
							for _, sh := range shareholders {
								subject := "Elezioni societa' " + cmp.Name
								content := fmt.Sprintf("Le elezioni per la societa' "+cmp.Name+" sono state vinte da "+winner.Name+" con %d voti/azione su %d.", maxvotes, totalvotes)
								report := &Report{PlayerID: sh.OwnerID, Date: endturn, Subject: subject, Content: content}
								if err := tx.Create(report); err.Error != nil {
									panic(err.Error)
								}

							}
						}

						if err := tx.Delete(&ElectionVote{}, "company_id = ?", cmp.ID); err.Error != nil {
							panic(err.Error)
						}

						cmp.CEOExpiration += opt.CEODuration
					}

					if err := tx.Save(cmp); err.Error != nil {
						panic(err.Error)
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

					if err := tx.Create(report); err.Error != nil {
						panic(err.Error)
					}
				}

				if err := tx.Model(&Player{}).Update("action_points", opt.PlayerActionPoints); err.Error != nil {
					panic(err.Error)
				}

				if err := tx.Model(&Company{}).Update("action_points", opt.CompanyActionPoints); err.Error != nil {
					panic(err.Error)
				}

				opt.LastTurnCalculated = endturn
				opt.Turn += 1
			}

			lastcheckpoint = endturn
			endturn = endturn.Add(time.Duration(opt.TurnDuration) * time.Minute)
		}

		opt.LastCheckpoint = now
		if err := tx.Save(opt); err.Error != nil {
			panic(err.Error)
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
	renderHTML(w, 200, templates.ErrorPage(err, string(req), pID, string(debug.Stack())))
}

var AdminPass string

func main() {
	debug := flag.Bool("debug", true, "turn on debug facilities")
	addr := flag.String("addr", ":8080", "address:port to bind to")
	flag.StringVar(&AdminPass, "pass", "admin", "administrator password")

	flag.Parse()

	var err error

	db, err = gorm.Open("sqlite3", "impero.db?_busy_timeout=10000")
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
		&ChatMessage{}, &Company{}, &Share{}, &ElectionProposal{},
		&ElectionVote{}, &Rental{}, &ShareAuction{},
		&ShareAuctionParticipation{}, &TransferProposal{})

	binder = NewGorillaBinder()

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
	game.HandleFunc("/player/transfer/action/", GameMiddleware(TransferAction)).Name("player_transfer_action")

	game.HandleFunc("/chat/", GameMiddleware(GetChat)).Name("chat")
	game.HandleFunc("/chat/post/", GameMiddleware(PostChat)).Name("chat_post")

	game.HandleFunc("/message/inbox/", GameMiddleware(MessagesInbox)).Name("message_inbox")
	game.HandleFunc("/message/outbox/", GameMiddleware(MessagesOutbox)).Name("message_outbox")
	game.HandleFunc("/message/{id}", GameMiddleware(GetMessage)).Name("message")
	game.HandleFunc("/message/new/", GameMiddleware(NewMessagePost)).Name("message_new")

	game.HandleFunc("/report/all/", GameMiddleware(ReportsPage)).Name("report_all")
	game.HandleFunc("/report/{id}", GameMiddleware(ReportPage)).Name("report")
	game.HandleFunc("/report/delete/", GameMiddleware(DeleteReports)).Name("report")

	game.HandleFunc("/company/all/", GameMiddleware(Companies)).Name("company_all")
	game.HandleFunc("/company/{id}", GameMiddleware(GetCompany)).Name("company")
	game.HandleFunc("/company/new/", GameMiddleware(NewCompanyPost)).Name("company_new")
	game.HandleFunc("/company/addshare/", GameMiddleware(AddShare)).Name("company_addshare")
	game.HandleFunc("/company/buy/", GameMiddleware(BuyNode)).Name("company_buy")
	game.HandleFunc("/company/invest/", GameMiddleware(InvestNode)).Name("company_invest")
	game.HandleFunc("/company/election/proposal/", GameMiddleware(NewElectionProposal)).Name("company_election_proposal")
	game.HandleFunc("/company/election/vote/", GameMiddleware(SetElectionVote)).Name("company_election_vote")

	game.HandleFunc("/bid/share/", GameMiddleware(BidShare)).Name("bid_share")
	game.HandleFunc("/map/", GameMiddleware(GetMap)).Name("map")

	logger.Println(http.ListenAndServe(*addr, router))
}
