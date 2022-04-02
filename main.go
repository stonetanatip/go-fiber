package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go/v4"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

const jwtSecret = "secret"

var db *sqlx.DB
var err error

func main() {

	db, err = sqlx.Open("mysql", "root:1234@tcp(localhost:3306)/user")
	if err != nil {
		panic(err)
	}
	app := fiber.New()

	app.Use("/hello", jwtware.New(jwtware.Config{
		SigningMethod: "HS256",
		SigningKey:    []byte(jwtSecret),
		SuccessHandler: func(c *fiber.Ctx) error {
			return c.Next()
		},
		ErrorHandler: func(c *fiber.Ctx, e error) error {
			return fiber.ErrUnauthorized
		},
	}))
	app.Post("/signup", Signup)
	app.Post("/login", Login)
	app.Get("/hello", Hello)

	app.Listen(":8000")

}

//curl localhost:8000/signup -d '{"username":"stone", "password":"1234"}' -H content-type:application/json -i
func Signup(c *fiber.Ctx) error {
	request := SignupRequest{}
	err := c.BodyParser(&request)
	if err != nil {
		return err
	}

	if request.Username == "" || request.Password == "" {
		return fiber.ErrUnprocessableEntity
	}

	password, err := bcrypt.GenerateFromPassword([]byte(request.Password), 10)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	query := "insert user (username, password) values (?, ?)"
	result, err := db.Exec(query, request.Username, string(password))
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	user := User{
		Id:       int(id),
		Username: request.Username,
		Password: string(password),
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

//curl localhost:8000/login -d '{"username":"stone", "password":"1234"}' -H content-type:application/json -i
func Login(c *fiber.Ctx) error {
	loginRequest := LoginRequest{}
	err := c.BodyParser(&loginRequest)
	if err != nil {
		return err
	}

	if loginRequest.Username == "" || loginRequest.Password == "" {
		return fiber.ErrUnprocessableEntity
	}

	user := User{}
	query := "select id, username, password from user where username=?"
	err = db.Get(&user, query, loginRequest.Username)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Incorrect username or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Incorrect username or password")
	}

	cliams := jwt.StandardClaims{
		Issuer: strconv.Itoa(user.Id),
		ExpiresAt: &jwt.Time{
			Time: time.Now().Add(time.Hour * 24),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, cliams)
	token, err := jwtToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.JSON(fiber.Map{
		"jwtToken": token,
	})
}

//curl localhost:8000/hello -H "Authorization:Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiI2In0.G99BarFVhIcVGJLn4p5op-aJjQcCeEkig1rALrYU1_g"
func Hello(c *fiber.Ctx) error {
	return c.SendString("Hello World !!!!!")
}

type User struct {
	Id       int    `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
	Password string `db:"password" json:"password"`
}

type SignupRequest struct {
	Username string `db:"username"`
	Password string `db:"password"`
}

type LoginRequest struct {
	Username string `db:"username"`
	Password string `db:"password"`
}

func Fiber() {
	app := fiber.New(
		fiber.Config{
			Prefork: true,
		},
	)

	//Middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("name", "stone")
		fmt.Println("Before")
		err := c.Next()
		fmt.Println("After")
		return err
	})

	app.Use(requestid.New())

	app.Use(cors.New(
		cors.Config{
			AllowOrigins: "*",
			AllowMethods: "*",
			AllowHeaders: "*",
		},
	))

	// app.Use(logger.New(logger.Config{
	// 	TimeZone: "Asia/Bangkok",
	// }))

	//GET
	//curl localhost:8000/hello -i
	app.Get("/hello", func(c *fiber.Ctx) error {
		name := c.Locals("name")

		return c.SendString(fmt.Sprintf("Hello %v !", name))

	})

	//POST
	//curl localhost:8000/hello -X POST -i
	app.Post("/hello", func(c *fiber.Ctx) error {
		return c.SendString("Hello Post!")
	})

	//Parameters
	//curl localhost:8000/hello/stone/tantip -i
	app.Get("/hello/:name/:surname", func(c *fiber.Ctx) error {
		name := c.Params("name")
		surname := c.Params("surname")
		return c.SendString("Hello " + name + " " + surname)
	})

	//ParamInt
	//curl localhost:8000/hello/1 -i
	app.Get("/hello/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {

		}
		return c.SendString(fmt.Sprintf("ID : %v", id))
	})

	//Query
	//curl "localhost:8000/query?name=stone" -i
	app.Get("/query", func(c *fiber.Ctx) error {
		name := c.Query("name")
		return c.SendString("Name :" + name)
	})

	//curl "localhost:8000/query2?id=1&name=stone" -i
	app.Get("/query2", func(c *fiber.Ctx) error {
		person := Person{}
		c.QueryParser(&person)
		return c.JSON(person)
	})

	//Wildcards
	//curl "localhost:8000/wildcards/stone/is/happy" -i
	app.Get("/wildcards/*", func(c *fiber.Ctx) error {
		wildCards := c.Params("*")
		return c.SendString(wildCards)
	})

	//Static file
	app.Static("/", "./wwwroot", fiber.Static{
		Index:         "index.html",
		CacheDuration: time.Second * 10,
	})

	//NewError
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "Content not found")
	})

	//Group
	v1 := app.Group("/v1", func(c *fiber.Ctx) error {
		c.Set("Version", "v1")
		return c.Next()
	})
	v1.Get("/hello", func(c *fiber.Ctx) error {
		return c.SendString("Hello v1 !")
	})

	v2 := app.Group("/v2", func(c *fiber.Ctx) error {
		c.Set("Version", "v2")
		return c.Next()
	})
	v2.Get("/hello", func(c *fiber.Ctx) error {
		return c.SendString("Hello v2 !")
	})

	//Mount
	//curl "localhost:8000/user/login" -i
	userApp := fiber.New()
	userApp.Get("/login", func(c *fiber.Ctx) error {
		return c.SendString("login")
	})

	app.Mount("/user", userApp)

	//Server
	// limit max connetion per IP
	app.Server().MaxConnsPerIP = 1
	app.Get("/server", func(c *fiber.Ctx) error {
		time.Sleep(time.Second * 10)
		return c.SendString("Server running !!!")
	})

	//Environment
	app.Get("/env", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"BaseURL":    c.BaseURL(),
			"Hostname":   c.Hostname(),
			"IP":         c.IP(),
			"IPs":        c.IPs(),
			"OrignalURL": c.OriginalURL(),
			"Path":       c.Path(),
			"Protocal":   c.Protocol(),
			"Subdomains": c.Subdomains(),
		})
	})

	//Body
	//curl localhost:8000/body -d '{"id":"1","name":"stone"}' -H content-type:application/json
	app.Post("/body", func(c *fiber.Ctx) error {
		fmt.Printf("IsJson: %v\n", c.Is("json"))
		fmt.Println(string(c.Body()))
		person := Person{}
		err := c.BodyParser(&person)
		if err != nil {
			return err
		}

		fmt.Println(person)
		return nil
	})

	app.Post("/body2", func(c *fiber.Ctx) error {
		fmt.Printf("IsJson: %v\n", c.Is("json"))
		fmt.Println(string(c.Body()))
		data := map[string]interface{}{}
		err := c.BodyParser(&data)
		if err != nil {
			return err
		}

		fmt.Println(data)
		return nil
	})

	app.Listen(":8000")
}

type Person struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
