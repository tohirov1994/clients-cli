package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tohirov1994/clients-core/pkg/core"
	_ "github.com/tohirov1994/database"
	"github.com/tohirov1994/terminal"
	"log"
	"os"
	"strconv"
	"strings"
)

var dataSource = "../database/db.sqlite"
//var dataSource = "github.com/tohirov1994/database/db.sqlite"

var IdClient int

var UserName string

func main() {
	log.Print("start application")
	log.Print("open db")
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		log.Fatalf("can't open db: %v", err)
	}
	defer func() {
		log.Print("close db")
		if err := db.Close(); err != nil {
			log.Fatalf("can't close db: %v", err)
		}
	}()
	err = core.Init(db)
	if err != nil {
		log.Fatalf("can't init db: %v", err)
	}
	terminal.Cleaner()
	fmt.Printf("\nATTENTION!!! THE PROGRAM WORKS EXCLUSIVELY WITH LATIN ALPHABET\n\n")
	fmt.Println("Welcome!")
	log.Print("start operations loop")
	operationsLoop(db, BeforeAuth, unauthorizedOperationsLoop)
	log.Print("finish operations loop")
	log.Print("finish application")
}

func operationsLoop(db *sql.DB, commands string, loop func(db *sql.DB, cmd string) bool) {
	for {
		fmt.Printf(commands)
		var cmd string
		_, err := fmt.Scan(&cmd)
		if err != nil {
			log.Fatalf("Can't read input: %v\n\n", err)
		}
		if exit := loop(db, strings.TrimSpace(cmd)); exit {
			return
		}
	}
}

func handleLogin(db *sql.DB) (okLogin bool, IdClient int, errSignIn error) {
	fmt.Println("Please enter Your identifications data")
	fmt.Print("Your Login: ")
	_, errSignIn = fmt.Scan(&UserName)
	if errSignIn != nil {
		return false, 0, errSignIn
	}
	var password string
	fmt.Print("Your Password: ")
	_, errSignIn = fmt.Scan(&password)
	if errSignIn != nil {
		return false, 0, errSignIn
	}
	IdClient, okLogin, errSignIn = core.SignIn(UserName, password, db)
	if errSignIn != nil {
		return false, 0, errSignIn
	}
	return okLogin, IdClient, errSignIn
}

func unauthorizedOperationsLoop(db *sql.DB, cmd string) (exit bool) {
	switch cmd {
	case "1":
		ok, authId, err := handleLogin(db)
		if err != nil {
			log.Printf("I can't execute authorization: %v", err)
			return true
		}
		if !ok {
			terminal.Cleaner()
			fmt.Println("Error, entered data is wrong. Try again.")
			return false
		}
		IdClient = authId
		fmt.Printf("\nWelcome: %s\n\n", UserName)
		operationsLoop(db, afterAuth, authorizedOperationsLoop)
	case "2":
		fmt.Printf("Available ATMs at addresses: \n")
		ATMs, err := core.ATMsGet(db)
		if err != nil {
			log.Printf("I can't get the ATM data: %v", err)
			return true
		} else {
			atmPrint(ATMs)
			fmt.Printf("\n")
			return false
		}
	case "q":
		return true
	default:
		fmt.Printf("You entered the wrong command: %s\n\n", cmd)
	}
	return false
}

func authorizedOperationsLoop(db *sql.DB, cmd string) (exit bool) {
	switch cmd {
	case "1":
		fmt.Printf("Your available Cards: \n")
		cards, err := core.CardsGet(IdClient, db)
		if err != nil {
			log.Printf("I can't get the your cards: %v\n\n", err)
			return true
		} else {
			myCardPrint(cards)
			fmt.Printf("#####################\n\n")
			return false
		}
	case "2":
		err := transfer(IdClient, db)
		if err != nil {
			fmt.Printf("Transfers canceled. Please check entered data!\n\n")
		} else {
			fmt.Printf("Transfers was successful.\n\n")
		}
	case "3":
		err := payService(IdClient, db)
		if err != nil {
			fmt.Printf("Payment canceled. Please check the entered data!\n\n")
		} else {
			fmt.Printf("Payment was successful.\n\n")
		}
	case "4":
		fmt.Printf("Available ATMs at addresses: \n")
		ATMs, err := core.ATMsGet(db)
		if err != nil {
			log.Printf("I can't get the ATM data: %v", err)
			return true
		} else {
			atmPrint(ATMs)
		}
	case "q":
		return true
	default:
		fmt.Printf("You entered the wrong command: %s\n", cmd)
	}
	return false
}

func myCardPrint(cards [] core.Card) {
	for _, card := range cards {
		fmt.Printf(
			"#####################\nid: %d\nPAN: %d\nPIN: %d\nCard account: %d\nHolderName: %s\nCVV: %d\nValidity(MMYY): %d\n",
			card.Id, card.PAN, card.PIN, card.Balance, card.HolderName, card.CVV, card.Validity,
		)
	}
}

func transfer(id int, db *sql.DB) (errTransact error) {
	id = IdClient
	cardAmount, errTransact := core.GetTransferCard(id, db)
	if cardAmount < 1 {
		fmt.Printf("You need a card to transfer, the count of your cards: %d", cardAmount)
		os.Exit(0)
	}
	if cardAmount == 1 {
		fmt.Println("You have one card, it will be used by default")
		curBalance, _ := core.GetCurrentBalanceClientId(id, db)
		fmt.Printf("Your card account is %d money available.\n", curBalance)
		if curBalance <= 0 {
			fmt.Printf("Please, replenish your account.\n")
			os.Exit(0)
		}
		fmt.Printf("You can transfer account to another account, with helpful PAN.\n")
		fmt.Printf("ATTENTION!!! Enter the card data correctly!\n")
		fmt.Printf("Enter receiver PAN: \n")
		var tmpPAN string
		_, errTransact = fmt.Scan(&tmpPAN)
		if errTransact != nil {
			return
		}
		var length int64
		length = int64(len(tmpPAN))
		if length != 16 {
			fmt.Printf("Check PAN! and try again.\n")
			os.Exit(0)
		}
		var panClient int64
		panClient, _ = strconv.ParseInt(tmpPAN, 10, 64)
		pan, err := core.CheckPan(panClient, db)
		if err != nil {
			fmt.Printf("The entered PAN is invalid, please enter the correct PAN.\n")
			os.Exit(0)
		}
		panClient = pan
		fmt.Printf("Enter transfer sum: (MAX transfer one million)\n")
		var amountTransact int
		_, errTransact = fmt.Scan(&amountTransact)
		if errTransact != nil {
			return errTransact
		}
		if amountTransact <= 0 {
			fmt.Printf("Please enter correctly sum!\n")
			os.Exit(0)
		}
		if amountTransact > 1_000_000 {
			fmt.Printf("You try transfer more max account.\n")
			os.Exit(0)
		}
		if amountTransact > curBalance {
			fmt.Printf("You enter incorrect sum, Your account: %d, query sum for transfer: %d\n", curBalance, amountTransact)
			os.Exit(0)
		}
		_, errTransact := core.OneCard(panClient, id, amountTransact, db)
		if errTransact != nil {
			fmt.Printf("Transfer of funds was canceled due to technical reasons, please correct valid data!: %v", errTransact)
			os.Exit(0)
		}
		fmt.Printf("Sender ID card: %d, receiver card: %d, account transfer: %d, successfully.\n", id, panClient, amountTransact)
	}
	if cardAmount > 1 {
		fmt.Println("You have a lot cards")
		fmt.Println("You will can use this cards:")
		fmt.Printf("Your available Cards: \n")
		cards, errTransact := core.CardsGet(id, db)
		if errTransact != nil {
			log.Printf("I can't get the your cards: %v\n\n", errTransact)
			os.Exit(0)
		}
		myCardPrint(cards)
		fmt.Printf("#####################\n\n")
		fmt.Println("Choose the card by PAN")
		fmt.Printf("Enter PAN of your card: \n")
		var tmpPAN string
		_, errTransact = fmt.Scan(&tmpPAN)
		if errTransact != nil {
			return errTransact
		}
		var length int64
		length = int64(len(tmpPAN))
		if length != 16 {
			fmt.Printf("Check length PAN! and try again.\n")
			os.Exit(0)
		}
		panMyPAN, _ := strconv.ParseInt(tmpPAN, 10, 64)
		cardSelected, errTransact := core.SelectCards(id, panMyPAN, db)
		if errTransact != nil {
			fmt.Printf("Check currectly PAN! and try again.\n")
			os.Exit(0)
		}
		curBalance, _ := core.GetCurrentBalanceClientPAN(cardSelected, db)
		fmt.Printf("Your card account is %d money available.\n", curBalance)
		if curBalance <= 0 {
			fmt.Printf("Please, replenish your account.\n")
			os.Exit(0)
		}
		fmt.Printf("You can transfer account to another account, with helpful PAN.\n")
		fmt.Printf("ATTENTION!!! Enter the card data correctly!\n")
		fmt.Printf("Enter receiver PAN: \n")
		_, errTransact = fmt.Scan(&tmpPAN)
		if errTransact != nil {
			return errTransact
		}
		length = int64(len(tmpPAN))
		if length != 16 {
			fmt.Printf("Check PAN! and try again.\n")
			os.Exit(0)
		}
		var panClient int64
		panClient, _ = strconv.ParseInt(tmpPAN, 10, 64)
		pan, err := core.CheckPan(panClient, db)
		if err != nil {
			fmt.Printf("The entered PAN is invalid, please enter the correct PAN.\n")
			os.Exit(0)
		}
		panClient = pan
		fmt.Printf("Enter transfer sum: (MAX transfer one million)\n")
		var amountTransact int
		_, errTransact = fmt.Scan(&amountTransact)
		if errTransact != nil {
			return errTransact
		}
		if amountTransact <= 0 {
			fmt.Printf("Please enter correctly sum!\n")
			os.Exit(0)
		}
		if amountTransact > 1_000_000 {
			fmt.Printf("You try transfer more max account.\n")
			os.Exit(0)
		}
		if amountTransact > curBalance {
			fmt.Printf("You enter incorrect sum, Your account: %d, query sum for transfer: %d\n", curBalance, amountTransact)
			os.Exit(0)
		}
		_, errTransact = core.MoreCard(cardSelected, panClient, amountTransact, db)
		if errTransact != nil {
			fmt.Printf("Transfer was canceled due to technical reasons, please check correct data!: %v", errTransact)
			os.Exit(0)
		}
		fmt.Printf("Sender ID card: %d, receiver card: %d, account transfer: %d, successfully.\n", IdClient, panClient, amountTransact)
	}
	return nil
}

func atmPrint(ATMs [] core.Atm) {
	for _, atm := range ATMs {
		fmt.Printf(
			"Id: %d, City: %s, District: %s, Street: %s\n",
			atm.Id, atm.City, atm.District, atm.Street,
		)
	}
}

func payService(payerId int, db *sql.DB) (err error) {
	payerId = IdClient
	cardAmount, err := core.GetTransferCard(payerId, db)
	if cardAmount < 1 {
		fmt.Printf("You need a card for pay services, the count of your cards: %d", cardAmount)
		os.Exit(0)
	}
	if cardAmount == 1 {
		fmt.Println("You have one card, it will be used by default")
		BalancePayer, _ := core.GetCurrentBalanceClientId(payerId, db)
		fmt.Printf("Your card account is %d money available.\n", BalancePayer)
		if BalancePayer <= 0 {
			fmt.Printf("Please, replenish your account.\n")
			os.Exit(0)
		}
		fmt.Printf("You can pay of services.\n")
		fmt.Printf("Available services: \n")
		services, err := core.GetAllService(db)
		if err != nil {
			log.Printf("I can't get the services data: %v", err)
			os.Exit(0)
		}
		PrintServices(services)
		fmt.Print("Enter Name of service: ")
		var nameService string
		_, err = fmt.Scan(&nameService)
		if err != nil {
			return err
		}
		nameService = strings.ToLower(nameService)
		serviceName, err := core.CheckServiceName(nameService, db)
		if err != nil {
			fmt.Printf("Enter correct data: %v", err)
			os.Exit(0)
		}
		fmt.Printf("Enter pay sum: (MAX pay for service one million)\n")
		var sum int
		_, err = fmt.Scan(&sum)
		if err != nil {
			return err
		}
		if sum <= 0 {
			fmt.Printf("Please enter correctly sum!\n")
			os.Exit(0)
		}
		if sum > 1_000_000 {
			fmt.Printf("You try pay more maximum.\n")
			os.Exit(0)
		}
		if sum > BalancePayer {
			fmt.Printf("You enter incorrect sum of service, Your account: %d, query sum for pay of service: %d\n", BalancePayer, sum)
			os.Exit(0)
		}
		_, err = core.ServicesPayOneCard(serviceName, payerId, sum, db)
		if err != nil {
			return err
		}
		fmt.Printf("You paid from card %d, service: %s, on sum %d, successfully.\n", payerId, serviceName, sum)
	}
	if cardAmount > 1 {
		fmt.Println("You have a lot cards")
		fmt.Println("You will can use this cards:")
		fmt.Printf("Your available Cards: \n")
		cards, errTransact := core.CardsGet(payerId, db)
		if errTransact != nil {
			log.Printf("I can't get the your cards: %v\n\n", errTransact)
			os.Exit(0)
		}
		myCardPrint(cards)
		fmt.Printf("#####################\n\n")
		fmt.Println("Choose the card by PAN")
		fmt.Printf("Enter PAN of your card: \n")
		var tmpPAN string
		_, errTransact = fmt.Scan(&tmpPAN)
		if errTransact != nil {
			return errTransact
		}
		var length int64
		length = int64(len(tmpPAN))
		if length != 16 {
			fmt.Printf("Check length PAN! and try again.\n")
			os.Exit(0)
		}
		panMyPAN, _ := strconv.ParseInt(tmpPAN, 10, 64)
		cardSelected, errTransact := core.SelectCards(payerId, panMyPAN, db)
		if errTransact != nil {
			fmt.Printf("Check currectly PAN! and try again.\n")
			os.Exit(0)
		}
		BalancePayer, _ := core.GetCurrentBalanceClientPAN(cardSelected, db)
		fmt.Printf("Your card account is %d money available.\n", BalancePayer)
		if BalancePayer <= 0 {
			fmt.Printf("Please, replenish your account.\n")
			os.Exit(0)
		}
		fmt.Printf("You can pay of services.\n")
		fmt.Printf("Available services: \n")
		services, err := core.GetAllService(db)
		if err != nil {
			log.Printf("I can't get the services data: %v", err)
			os.Exit(0)
		}
		PrintServices(services)
		fmt.Print("Enter Name of service: ")
		var nameService string
		_, err = fmt.Scan(&nameService)
		if err != nil {
			return err
		}
		nameService = strings.ToLower(nameService)
		serviceName, err := core.CheckServiceName(nameService, db)
		if err != nil {
			fmt.Printf("Enter correct data: %v", err)
			os.Exit(0)
		}
		fmt.Printf("Enter pay sum: (MAX pay for service one million)\n")
		var sum int
		_, err = fmt.Scan(&sum)
		if err != nil {
			return err
		}
		if sum <= 0 {
			fmt.Printf("Please enter correctly sum!\n")
			os.Exit(0)
		}
		if sum > 1_000_000 {
			fmt.Printf("You try pay more maximum.\n")
			os.Exit(0)
		}
		if sum > BalancePayer {
			fmt.Printf("You enter incorrect sum of service, Your account: %d, query sum for pay of service: %d\n", BalancePayer, sum)
			os.Exit(0)
		}
		_, err = core.ServicesPayMoreCard(serviceName, cardSelected, sum, db)
		if err != nil {
			return err
		}
		fmt.Printf("You paid from card %d, service: %s, on sum %d, successfully.\n", cardSelected, serviceName, sum)
	}
	return nil
}

func PrintServices(services [] core.ServicesStruct) {
	for _, service := range services {
		fmt.Printf(
			"service Id: %d, service Name: %s\n", service.Id, service.Service,
		)
	}
}
