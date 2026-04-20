package symbols

import (
	"testing"
)

func TestExtractGo(t *testing.T) {
	src := `package main

import "fmt"

type UserService struct {
	db *DB
}

type Reader interface {
	Read(p []byte) (n int, err error)
}

func NewUserService(db *DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) GetUser(id int) (*User, error) {
	return s.db.Find(id)
}
`
	syms := Extract(src, "main.go")

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	if len(kinds["class"]) != 1 || kinds["class"][0] != "UserService" {
		t.Errorf("expected struct UserService, got %v", kinds["class"])
	}
	if len(kinds["interface"]) != 1 || kinds["interface"][0] != "Reader" {
		t.Errorf("expected interface Reader, got %v", kinds["interface"])
	}
	if len(kinds["function"]) < 2 {
		t.Errorf("expected at least 2 functions, got %v", kinds["function"])
	}
}

func TestExtractTypeScript(t *testing.T) {
	src := `export interface Config {
  port: number;
}

export class Server {
  constructor(private config: Config) {}

  start(): void {
    console.log("starting");
  }
}

export function createServer(config: Config): Server {
  return new Server(config);
}

export const handler = async (req: Request) => {
  return new Response("ok");
};

export type ServerOptions = {
  port: number;
};

export enum Status {
  Active,
  Inactive,
}
`
	syms := Extract(src, "server.ts")

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	if len(kinds["interface"]) != 1 || kinds["interface"][0] != "Config" {
		t.Errorf("expected interface Config, got %v", kinds["interface"])
	}
	if len(kinds["class"]) != 1 || kinds["class"][0] != "Server" {
		t.Errorf("expected class Server, got %v", kinds["class"])
	}
	if len(kinds["function"]) < 2 {
		t.Errorf("expected at least 2 functions, got %v", kinds["function"])
	}
	if len(kinds["type"]) != 1 || kinds["type"][0] != "ServerOptions" {
		t.Errorf("expected type ServerOptions, got %v", kinds["type"])
	}
	if len(kinds["enum"]) != 1 || kinds["enum"][0] != "Status" {
		t.Errorf("expected enum Status, got %v", kinds["enum"])
	}
}

func TestExtractPython(t *testing.T) {
	src := `class UserManager:
    def __init__(self, db):
        self.db = db

    def get_user(self, user_id):
        return self.db.find(user_id)

async def fetch_data(url):
    pass

def process(items):
    return [i for i in items]
`
	syms := Extract(src, "app.py")

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	if len(kinds["class"]) != 1 || kinds["class"][0] != "UserManager" {
		t.Errorf("expected class UserManager, got %v", kinds["class"])
	}
	if len(kinds["function"]) < 2 {
		t.Errorf("expected at least 2 functions, got %v", kinds["function"])
	}
}

func TestExtractRust(t *testing.T) {
	src := `pub struct Config {
    port: u16,
}

pub trait Handler {
    fn handle(&self, req: Request) -> Response;
}

pub enum Status {
    Active,
    Inactive,
}

impl Config {
    pub fn new(port: u16) -> Self {
        Config { port }
    }
}

pub async fn serve(config: Config) {
    // ...
}
`
	syms := Extract(src, "lib.rs")

	kinds := map[string][]string{}
	for _, s := range syms {
		kinds[s.Kind] = append(kinds[s.Kind], s.Name)
	}

	if len(kinds["class"]) < 2 { // struct + impl
		t.Errorf("expected at least 2 class entries (struct+impl), got %v", kinds["class"])
	}
	if len(kinds["interface"]) != 1 || kinds["interface"][0] != "Handler" {
		t.Errorf("expected trait Handler, got %v", kinds["interface"])
	}
	if len(kinds["enum"]) != 1 || kinds["enum"][0] != "Status" {
		t.Errorf("expected enum Status, got %v", kinds["enum"])
	}
	if len(kinds["function"]) < 1 {
		t.Errorf("expected at least 1 function, got %v", kinds["function"])
	}
}

func TestFormat(t *testing.T) {
	syms := []Symbol{
		{Name: "Foo", Kind: "class", Line: 1, Signature: "class Foo {"},
		{Name: "bar", Kind: "function", Line: 5, Signature: "func bar() {"},
	}
	out := Format(syms)
	if out == "(no symbols found)" {
		t.Error("expected formatted output, got empty")
	}
}

func TestFormatEmpty(t *testing.T) {
	out := Format(nil)
	if out != "(no symbols found)" {
		t.Errorf("expected '(no symbols found)', got %q", out)
	}
}
