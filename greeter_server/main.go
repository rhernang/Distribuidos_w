/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	port                = ":50051"
	addressPozoGRPC     = "localhost:50053"
	addressNameNodeGRPC = "localhost:50054"
)

type server struct{ pb.UnimplementedGreeterServer }

//VARIABLES GENERALES
var MaxPlayers int = 3
var NumberOfPlayers int = 0
var NumberOfPlayersReady int = 0
var ListOfLivePlayers = [3]string{"y", "y", "y"}

//VARIABLES JUEGO 1
var numberG1 int = 0

//VARIABLES JUEGO 2
var RPlayerEliminated string = "-"
var TotalG2T1 int = 0
var TotalG2T2 int = 0
var eleccG2 int = 0
var TeamPlayers = [3]string{"-", "-", "-"}
var LoseTeam string = "0"

//VARIABLE JUEGO3
var numberG3 int = 0
var PairPlayers = [3]int{-1, -1, -1}
var AnswerPlayers = [3]int{-1, -1, -1}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

//CONEXIONES
func grpcChannel(ipAdress string, message string) string {
	conn, err := grpc.Dial(ipAdress, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: message})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	return r.GetMessage()
}

func rabbitmqChannel(message string) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()
	q, err := ch.QueueDeclare("hello", false, false, false, false, nil)
	failOnError(err, "Failed to declare a queue")

	body := message + " 1 100"
	err = ch.Publish("", q.Name, false, false, amqp.Publishing{ContentType: "text/plain", Body: []byte(body)})
	failOnError(err, "Failed to publish a message")
}

func ListenMessage() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

//ENVIAR MENSAJE A LOS JUGADORES
//IDPLAYER := -1 para enviar a todos
func SendMessageToPlayers(msgLider string, IDplayer int) {
	var message string
	var UserToEliminated int = IDplayer

	if msgLider == "R" {
		message = "Ready"
	}
	if msgLider == "1" || msgLider == "2" || msgLider == "3" {
		message = "G" + msgLider
	}
	if msgLider == "D" || msgLider == "DT" {
		NumberOfPlayers = NumberOfPlayers - 1
		if msgLider == "D" {
			UserToEliminated = A_IDplayer()
		}
		_ = SendMessageToPozo("", strconv.FormatInt(int64(UserToEliminated), 10))
		message = "death " + strconv.FormatInt(int64(UserToEliminated), 10)
	}

	for i := 0; i < len(ListOfLivePlayers); i++ {
		if ListOfLivePlayers[i] == "y" {
			_ = grpcChannel("localhost:"+strconv.FormatInt(int64(50060+i+1), 10), message)
		}
	}
}

//MANDAR MENSAJES AL POZO
func SendMessageToPozo(msg string, player string) string {
	if msg == "val" {
		return grpcChannel(addressPozoGRPC, msg)
	}
	rabbitmqChannel(player)
	return ""
}

func SendMessageToNameNode(msg string) string {
	return grpcChannel(addressNameNodeGRPC, msg)
}

// ESCUCHAR MENSAJES DE LOS JUGADORES
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {

	//INGRESAR AL JUEGO
	if in.GetName() == "yes" {
		NumberOfPlayers = NumberOfPlayers + 1
		fmt.Println("Esperando jugadores ", NumberOfPlayers, "/", MaxPlayers)
		return &pb.HelloReply{Message: strconv.FormatInt(int64(NumberOfPlayers), 10)}, nil
	}

	//SEPARACION DEL MENSAJE
	text := strings.Split(in.GetName(), " ")
	//Game := text[1]
	//RondaJugada := text[2]
	IDplayer, _ := strconv.Atoi(text[3])
	Jugada, _ := strconv.Atoi(text[4])

	if text[4] != "R" && text[4] != "RD" && text[4] != "VP" {
		_ = SendMessageToNameNode(text[1] + " " + text[3] + " " + text[4])
	}

	if text[4] == "R" {
		if text[1] == "G2" && TeamPlayers[IDplayer-1] == LoseTeam {
			_ = SendMessageToPozo("", text[3])
			fmt.Println("Jugador " + text[3] + " ha muerto")

			ListOfLivePlayers[IDplayer-1] = "n"
			NumberOfPlayers = NumberOfPlayers - 1
			fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)

			return &pb.HelloReply{Message: "death"}, nil
		}
		if text[1] == "G3" {
			var aux int = FindPair(PairPlayers[IDplayer-1], IDplayer)
			if AnswerPlayers[IDplayer-1] > AnswerPlayers[aux] {
				_ = SendMessageToPozo("", text[3])
				fmt.Println("Jugador " + text[3] + " ha muerto")

				ListOfLivePlayers[IDplayer-1] = "n"
				NumberOfPlayers = NumberOfPlayers - 1
				fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)

				return &pb.HelloReply{Message: "death"}, nil
			}
		}

		NumberOfPlayersReady = NumberOfPlayersReady + 1
		fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)
		return &pb.HelloReply{Message: "live"}, nil
	}

	if text[4] == "RD" {
		if RPlayerEliminated == text[3] {
			_ = SendMessageToPozo("", text[3])
			fmt.Println("Jugador " + text[3] + " ha muerto")

			ListOfLivePlayers[IDplayer-1] = "n"
			NumberOfPlayers = NumberOfPlayers - 1
			fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)

			return &pb.HelloReply{Message: "death"}, nil
		}

		NumberOfPlayersReady = NumberOfPlayersReady + 1
		fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)
		return &pb.HelloReply{Message: "live"}, nil
	}

	if text[4] == "VP" {
		return &pb.HelloReply{Message: SendMessageToPozo("val", "")}, nil
	}

	if text[1] == "G1" {
		if Jugada >= numberG1 || text[4] == "death" {
			_ = SendMessageToPozo("", text[3])
			fmt.Println("Jugador " + text[3] + " ha muerto")

			ListOfLivePlayers[IDplayer-1] = "n"
			NumberOfPlayers = NumberOfPlayers - 1
			fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)
			return &pb.HelloReply{Message: "death"}, nil
		}

		NumberOfPlayersReady = NumberOfPlayersReady + 1
		fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)
		return &pb.HelloReply{Message: "live"}, nil
	}

	if text[1] == "G2" {
		if TeamPlayers[IDplayer-1] == "1" {
			TotalG2T1 = TotalG2T1 + Jugada
		}
		if TeamPlayers[IDplayer-1] == "2" {
			TotalG2T2 = TotalG2T2 + Jugada
		}
		NumberOfPlayersReady = NumberOfPlayersReady + 1
		fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)
		return &pb.HelloReply{Message: "wait"}, nil
	}

	if text[1] == "G3" {
		AnswerPlayers[IDplayer-1] = Jugada
		NumberOfPlayersReady = NumberOfPlayersReady + 1
		fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)
		return &pb.HelloReply{Message: "wait"}, nil
	}

	return nil, nil
}

//MENUS Y OTROS
func LivePlayers() {
	for i := 0; i < len(ListOfLivePlayers); i++ {
		if ListOfLivePlayers[i] == "y" {
			fmt.Println("Jugador ", i+1, "ha sobrevivido")
		}
	}
}

func Menu() {
	fmt.Println("***************************************************")
	fmt.Println("Elija 1 para comenzar el juego Luz Roja Luz Verde")
	fmt.Println("Elija 2 para comenzar el juego Tirar la cuerda")
	fmt.Println("Elija 3 para comenzar el juego Todo o nada")
	fmt.Println("Elija 4 para ver el valor del pozo")
	fmt.Println("***************************************************")
}

func A_IDplayer() int {
	IDAleatorio := rand.Intn(NumberOfPlayers) + 1
	for i := 0; i < len(ListOfLivePlayers); i++ {
		if ListOfLivePlayers[i] == "y" {
			IDAleatorio = IDAleatorio - 1
		}
		if IDAleatorio == 0 && ListOfLivePlayers[i] == "y" {
			return i + 1
		}
	}
	return 0
}

func FindPair(t int, p int) int {
	var aux int = 0
	for i := 0; i < len(ListOfLivePlayers); i++ {
		if PairPlayers[i] == t && (p-1) != i {
			aux = i
		}
	}
	return aux
}

func DefineTeamsG2() {
	var aux int = 0
	for i := 0; i < len(ListOfLivePlayers); i++ {
		if aux == 0 && ListOfLivePlayers[i] == "y" {
			TeamPlayers[i] = "1"
		}
		if aux == 1 && ListOfLivePlayers[i] == "y" {
			TeamPlayers[i] = "2"
		}
		aux = (aux + 1) % 2
	}
}

func DefineTeamsG3() {
	var aux int = 0
	for i := 0; i < len(ListOfLivePlayers); i++ {
		if ListOfLivePlayers[i] == "y" {
			PairPlayers[i] = aux
		}
		aux = (aux + 1) % (NumberOfPlayers / 2)
	}
}

//MAIN
func main() {

	go ListenMessage()

	forever := make(chan bool)
	var elecc string

	fmt.Println("Esperando jugadores ", NumberOfPlayers, "/", MaxPlayers)
	for {
		if NumberOfPlayers == MaxPlayers {
			break
		}
	}

	for {
		Menu()
		fmt.Scanf("%s", &elecc)
		SendMessageToPlayers(elecc, 0)

		//PRIMER JUEGO
		if elecc == "1" {
			SendMessageToPlayers("R", 0)
			fmt.Println("Primer juego")
			fmt.Println("Debe elegir 4 numeros entre 6 y 10")
			for round := 0; round < 4; round++ {
				fmt.Println("Elija un numero")
				fmt.Scanf("%d", &numberG1)

				SendMessageToPlayers("R", 0)

				fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)
				for {
					if NumberOfPlayersReady == NumberOfPlayers {
						break
					}
				}

				if NumberOfPlayers == 0 {
					fmt.Println("Todos los jugadores murieron")
					break
				}
				NumberOfPlayersReady = 0
			}

			fmt.Println("Juego finalizado")
			fmt.Println("Jugadores sobrevivientes ", NumberOfPlayersReady)
			LivePlayers()
		}

		if elecc == "2" {

			NumberOfPlayersReady = 0
			SendMessageToPlayers("R", 0)
			if NumberOfPlayers%2 == 1 && NumberOfPlayers != 1 {
				RPlayerEliminated = strconv.FormatInt(int64(A_IDplayer()), 10)
				SendMessageToPlayers("R", 0)
				for {
					if NumberOfPlayersReady == NumberOfPlayers {
						break
					}
				}
			}
			NumberOfPlayersReady = 0

			DefineTeamsG2()
			fmt.Println("Segundo juego")
			fmt.Println("Debe elegir un numero entre 1 y 4")

			fmt.Scanf("%d", &eleccG2)
			eleccG2 = eleccG2 % 2

			SendMessageToPlayers("R", 0)

			for {
				if NumberOfPlayersReady == NumberOfPlayers {
					break
				}
			}
			NumberOfPlayersReady = 0

			if TotalG2T1%2 == TotalG2T2%2 && TotalG2T1 != eleccG2 {
				aux := rand.Intn(2)
				if aux == 0 {
					LoseTeam = "1"
				}
				if aux == 1 {
					LoseTeam = "2"
				}
			}

			if TotalG2T1%2 != eleccG2 {
				LoseTeam = "1"
			}
			if TotalG2T2%2 != eleccG2 {
				LoseTeam = "2"
			}

			fmt.Println("Equipo perdedor ", LoseTeam)
			SendMessageToPlayers("R", 0)
			for {
				if NumberOfPlayersReady == NumberOfPlayers {
					break
				}
			}

			fmt.Println("Juego finalizado")
			fmt.Println("Jugadores sobrevivientes ", NumberOfPlayersReady)
			LivePlayers()
			NumberOfPlayersReady = 0
			//SendMessageToPlayers("R", 0)
		}

		if elecc == "3" {

			NumberOfPlayersReady = 0
			SendMessageToPlayers("R", 0)
			if NumberOfPlayers%2 == 1 && NumberOfPlayers != 1 {
				RPlayerEliminated = strconv.FormatInt(int64(A_IDplayer()), 10)
				SendMessageToPlayers("R", 0)
				for {
					if NumberOfPlayersReady == NumberOfPlayers {
						break
					}
				}
			}
			NumberOfPlayersReady = 0

			DefineTeamsG3()

			fmt.Println("Tercer juego")
			fmt.Println("Debe elegir un numero entre 1 y 10")

			fmt.Println("Elija un numero")
			fmt.Scanf("%d", &numberG3)

			SendMessageToPlayers("R", 0)
			fmt.Println("Esperando jugadores", NumberOfPlayersReady, "/", NumberOfPlayers)
			for {
				if NumberOfPlayersReady == NumberOfPlayers {
					break
				}
			}
			NumberOfPlayersReady = 0

			for i := 0; i < len(ListOfLivePlayers); i++ {
				if ListOfLivePlayers[i] == "y" {
					if AnswerPlayers[i] >= numberG3 {
						AnswerPlayers[i] = AnswerPlayers[i] - numberG3
					}
					if AnswerPlayers[i] < numberG3 {
						AnswerPlayers[i] = numberG3 - AnswerPlayers[i]
					}
				}
			}

			SendMessageToPlayers("R", 0)
			for {
				if NumberOfPlayersReady == NumberOfPlayers {
					break
				}
			}

			fmt.Println("Juego finalizado")
			fmt.Println("Los jugadores ganadores son ", NumberOfPlayersReady)
			LivePlayers()
			NumberOfPlayersReady = 0
		}

		if elecc == "4" {
			aux := SendMessageToPozo("val", "")
			fmt.Println("Valor en el pozo: ", aux)
		}
	}
	<-forever
}
