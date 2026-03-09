package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Vladroon22/2FA/internal/core"
	otpservice "github.com/Vladroon22/2FA/internal/otp-service"
	"github.com/gorilla/mux"
)

type Server struct {
	serv *http.Server
	otp  *otpservice.OTPService
}

func NewServer() *Server {
	otp := otpservice.NewOTPService()
	s := &Server{
		serv: &http.Server{},
		otp:  otp,
	}
	go otp.Clean()
	return s
}

func (s *Server) handleAddAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Is string `json:"is"`
		Ac string `json:"ac"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	secret, _, err := core.GenerateSecretWithURI(req.Is, req.Ac)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	if err := s.otp.SaveOTP(now, req.Is, secret); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	///qrterminal.Generate(uri, qrterminal.M, w)
	w.Write([]byte(secret))
}

func (s *Server) Verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Is   string `json:"is"`
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	now := time.Now()
	if ok, err := s.otp.ValidateOTP(now, req.Code, req.Is); err != nil || !ok {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Write([]byte("Valid"))
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	cnfs := s.otp.GetHealth()
	json.NewEncoder(w).Encode(cnfs)
}

func main() {
	srv := NewServer()

	r := mux.NewRouter()
	r.HandleFunc("/send", srv.handleAddAccount).Methods("POST")
	r.HandleFunc("/verify", srv.Verify).Methods("POST")
	r.HandleFunc("/health", srv.Health).Methods("GET")
	srv.serv.Handler = r

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}
	srv.serv.Serve(l)
}
