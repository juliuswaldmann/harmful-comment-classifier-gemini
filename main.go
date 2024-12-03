package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"net/http"

	"encoding/json"

	"github.com/gorilla/mux"
)

var model *genai.GenerativeModel

//default timeout. may get overwritten with environment variables
var requestTimeout time.Duration = 30 * time.Second 

//default http listen address. may get overwritten with environment variables
var http_addr = "0.0.0.0:3343"

func main() {

    //allow setting of custom timeout.
    custom_request_Timeout, ok := os.LookupEnv("CUSTOM_REQUEST_TIMEOUT_SECONDS")
    if ok {
        ct, err := strconv.Atoi(custom_request_Timeout)
        if err == nil && ct > 0 {
            requestTimeout = time.Duration(ct) * time.Second
        } else {
            log.Printf(
                "setting of custom timeout failed. %v is not a valid value (has to be an postive integer). Default is used",
                custom_request_Timeout,
                )
        }
    }

    //allow setting custom http_listen_address
    custom_http_addr, ok := os.LookupEnv("HTTP_LISTEN_ADDR")
    if ok {
        http_addr = custom_http_addr
    }
    
    //connect the gemini client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

    //allow setting what model in the api is used.
    //defaults to gemini-1.5-flash
    gemini_model_name, ok := os.LookupEnv("CUSTOM_GEMINI_MODEL")
    if !ok {
        gemini_model_name = "gemini-1.5-flash"
    }
    model = client.GenerativeModel(gemini_model_name)


    //read systemprompt from prompt.txt and apply it to the model
    system_prompt, err := os.ReadFile("./prompt.txt")
	if err != nil {
		log.Fatal(err)
	}
    model.SystemInstruction = genai.NewUserContent(genai.Part(genai.Text(system_prompt)))


    //routing
    r := mux.NewRouter()
    r.Use(loggingMiddleware)

    r.HandleFunc("/block/{comment}", blockHandler).Methods("GET")
    r.HandleFunc("/block", blockJsonHandler).Methods("GET")
    

    srv := &http.Server{
        Handler:      r,
        Addr:         http_addr,

        WriteTimeout: requestTimeout,
        ReadTimeout:  requestTimeout,
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
        log.Printf("Internal Server Error: %v", err.Error())
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    result, err := AnalyzeText(req.Comment, model)
    if err != nil {
        log.Printf("Internal Server Error: %v", err.Error())
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    blockResponse := BlockResponse{
        Block: result, 
    }
    responseBytes, err := json.Marshal(blockResponse)
    if err != nil {
        log.Printf("Internal Server Error: %v", err.Error())
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(responseBytes)
}

func blockHandler(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    var comment string
    comment = params["comment"]

    result, err := AnalyzeText(comment, model)
    if err != nil {
        log.Printf("Internal Server Error: %v", err.Error())
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
    ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
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
