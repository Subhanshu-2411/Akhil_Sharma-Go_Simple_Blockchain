package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"time"
)

type Block struct {
	Position  int
	Data      BookCheckout
	TimeStamp string
	Hash      string
	PrevHash  string
}

func (block *Block) generateHash() {
	bytes, _ := json.Marshal(block.Data)

	data := string(block.Position) + block.TimeStamp + string(bytes) + block.PrevHash

	hash := sha256.New()
	hash.Write([]byte(data))
	block.Hash = hex.EncodeToString(hash.Sum(nil))
}

func (block *Block) validateHash(hash string) bool {
	block.generateHash()
	if block.Hash != hash {
		return false
	}
	return true
}

type BookCheckout struct {
	bookID       string `json:"book_id"`
	User         string `json:"user"`
	CheckoutDate string `json:"checkout_date"`
	IsGenesis    bool   `json:"is_genesis"`
}

type Book struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	PublishedDate string `json:"published_date"`
	ISBN          string `json:"isbn"`
}

type Blockchain struct {
	blocks []*Block
}

var blockchain *Blockchain

func (bc *Blockchain) AddBlock(data BookCheckout) {
	prevBlock := bc.blocks[len(bc.blocks)-1]
	block := CreateBlock(prevBlock, data)

	if validBlock(block, prevBlock) {
		bc.blocks = append(bc.blocks, block)
	}
}

func validBlock(block, prevBlock *Block) bool {
	if prevBlock.Hash != block.PrevHash {
		return false
	}

	if !block.validateHash(block.Hash) {
		return false
	}

	if prevBlock.Position+1 != block.Position {
		return false
	}

	return true
}

func CreateBlock(prevBlock *Block, checkoutItem BookCheckout) *Block {
	block := &Block{}
	block.Position = prevBlock.Position + 1
	block.TimeStamp = time.Now().String()
	block.PrevHash = prevBlock.Hash
	block.generateHash()

	return block
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, BookCheckout{IsGenesis: true})
}

func NewBlockChain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}

func main() {
	blockchain = NewBlockChain()
	r := mux.NewRouter()
	r.HandleFunc("/", getBlockchain).Methods("GET")
	r.HandleFunc("/", writeBlock).Methods("POST")
	r.HandleFunc("/new", newBook).Methods("POST")

	go func() {
		for _, block := range blockchain.blocks {
			fmt.Printf("Prev Hash: %x\n", block.PrevHash)
			bytes, _ := json.MarshalIndent(block.Data, "", " ")
			fmt.Printf("Data: %v\n", string(bytes))
			fmt.Printf("Hash: %x\n", block.Hash)
		}
	}()

	log.Println("Listening on Port 3000")

	log.Fatal(http.ListenAndServe(":3000", r))

}

func getBlockchain(writer http.ResponseWriter, request *http.Request) {
	jBytes, err := json.MarshalIndent(blockchain.blocks, "", " ")
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(writer).Encode(err)
		return
	}
	io.WriteString(writer, string(jBytes))
}

func writeBlock(writer http.ResponseWriter, request *http.Request) {
	var checkoutItem BookCheckout
	if err := json.NewDecoder(request.Body).Decode(&checkoutItem); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could Not Write Block: %v", err)
		writer.Write([]byte("Could Not Write block"))
	}

	blockchain.AddBlock(checkoutItem)
}

func newBook(writer http.ResponseWriter, request *http.Request) {
	var book Book

	if err := json.NewDecoder(request.Body).Decode(&book); err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not Create the Book: %v", err)
		writer.Write([]byte("Could Not Create new Book"))
		return
	}
	h := md5.New()

	io.WriteString(h, book.ISBN+book.PublishedDate)
	book.ID = fmt.Sprintf("%x", h.Sum(nil))

	response, err := json.MarshalIndent(book, "", " ")
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could Not Marshal Payload %v", err)
		writer.Write([]byte("Could Not Save Book Data"))
		return
	}
	writer.WriteHeader(http.StatusOK)
	writer.Write(response)

}
