package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"net/http"

	"github.com/gorilla/mux"
    "encoding/json"
)

var model *genai.GenerativeModel
var ctx context.Context

func main() {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	model = client.GenerativeModel("gemini-1.5-flash")

    data, err := os.ReadFile("./prompt.txt")
	if err != nil {
		log.Fatal(err)
	}
    system_prompt := string(data)
    model.SystemInstruction = genai.NewUserContent(genai.Part(genai.Text(system_prompt)))


    r := mux.NewRouter()
    r.Use(loggingMiddleware)

    r.HandleFunc("/block/{comment}", blockHandler).Methods("GET")
    r.HandleFunc("/block", blockJsonHandler).Methods("GET")
    
    http_addr := os.Getenv("HTTP_LISTEN_ADDR")

    srv := &http.Server{
        Handler:      r,
        Addr:         http_addr,

        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }

    log.Printf("running http server on %v\n", http_addr)
    log.Fatal(srv.ListenAndServe())
}


type BlockRequest struct {
    Comment string `json:"comment"`
}
type BlockResponse struct {
    Block bool `json:"block"`
}

func blockJsonHandler(w http.ResponseWriter, r *http.Request) {
    var req BlockRequest
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        w.Write([]byte(err.Error()))
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    result, err := AnalyzeText(req.Comment, model)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    blockResponse := BlockResponse{
        Block: result, 
    }
    responseBytes, err := json.Marshal(blockResponse)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    w.Write(responseBytes)
}

func blockHandler(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    var comment string
    comment = params["comment"]

    result, err := AnalyzeText(comment, model)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    if result == true {
        w.Write([]byte("\"True\""))
    } else {
        w.Write([]byte("\"False\""))
    }
}

func AnalyzeText(text string, model *genai.GenerativeModel) (result bool, err error){
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

	cs := model.StartChat()

	resp, err := cs.SendMessage(ctx, genai.Text(text))
	if err != nil {
        return result, err
	}

    response := strings.TrimSpace(fmt.Sprint(resp.Candidates[0].Content.Parts[0]))
    if response[len(response)-4:] == "True" {
        result = true
    } else if response[len(response)-5:] == "False" {
        result = false
    } else {
        return result, fmt.Errorf("LLM returned invalid response")
    }

    return 
}


func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Println(r.RequestURI)
        next.ServeHTTP(w, r)
    })
}
