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

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag" //"strings"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	defaultName = "world"
)

var (
	addr = flag.String("addr", "dist213.inf.santiago.usm.cl:50052", "the address to connect to")
	name = flag.String("name", defaultName, "Name to greet")
)

var JuegoActual string = ""
var IDplayer string = ""
var ReadyToPlay string = ""
var RoundGame = ""

//ESCUCHAR AL LIDER
type server struct{ pb.UnimplementedGreeterServer }

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	if in.GetName() == "G1" || in.GetName() == "G2" || in.GetName() == "G3" {
		JuegoActual = in.GetName()
	}
	if in.GetName() == "Ready" {
		ReadyToPlay = "Ready"
	}
	return &pb.HelloReply{Message: "recibido"}, nil
}

func grpcChannel(message string) string {
	conn, err := grpc.Dial("dist213.inf.santiago.usm.cl:50052", grpc.WithInsecure(), grpc.WithBlock())
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

func ListenInstr() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func ComunicacionLider(msg string) string {
	var message string = ":50052" + " " + JuegoActual + " " + RoundGame + " " + IDplayer + " " + msg

	//INDICARLE QUE VA A JUGAR
	if msg == "Begin" {
		return grpcChannel("yes")
	}

	//INDICARLE QUE EL JUGADOR PERDIO
	if msg == "death" {
		message = ":50052" + " " + JuegoActual + " " + RoundGame + " " + IDplayer + " death"
	}

	if msg == "resultado" {
		message = ":50052" + " " + JuegoActual + " " + RoundGame + " " + IDplayer + " R"
	}
	if msg == "RandomDeath" {
		message = ":50052" + " " + JuegoActual + " " + RoundGame + " " + IDplayer + " RD"
	}
	if msg == "ValPozo" {
		message = ":50052" + " " + JuegoActual + " " + RoundGame + " " + IDplayer + " VP"
	}

	r := grpcChannel(message)
	return r
}

func main() {

	var elecc string
	forever := make(chan bool)
	go ListenInstr()

	fmt.Println("Bienvenido al juego del calamar")
	fmt.Println("¿Desea jugar? yes/no")
	fmt.Scanf("%s", &elecc)

	if elecc != "yes" {
		return
	}
	IDplayer = ComunicacionLider("Begin")

	for {

		fmt.Println("Esperando ...")
		for {
			if ReadyToPlay == "Ready" {
				break
			}
		}
		ReadyToPlay = ""

		if JuegoActual == "G1" {
			var round int = 0
			var total int = 0
			var resp string
			var num string

			fmt.Println("Bienvenido al primer Juego: Luz verde Luz roja")
			fmt.Println("Debera elegir 4 veces un numero entre el 1 y el 10")
			fmt.Println("El objetivo es que la suma de sus numeros sea mayor o igual que 21")
			fmt.Println("Si usted elige un numero mayor o igual que el lider, quedara descalificado")

			for round < 4 {
				fmt.Println("Esperando eleccion del lider")
				for {
					if ReadyToPlay == "Ready" {
						break
					}
				}

				fmt.Println("Elija un numero")
				fmt.Scanf("%s", &num)
				aux, _ := strconv.Atoi(num)
				total = total + aux

				resp = ComunicacionLider(num)
				if resp == "death" {
					fmt.Println("Ha muerto")
					return
				}
				ReadyToPlay = ""
				round = round + 1
			}

			if total < 21 {
				fmt.Println("Ha muerto")
				_ = ComunicacionLider("death")
				return
			}
			fmt.Println("El valor del pozo actual es: ", ComunicacionLider("ValPozo"))
			ReadyToPlay = ""
			JuegoActual = ""
		}

		if JuegoActual == "G2" {

			for {
				if ReadyToPlay == "Ready" {
					resp := ComunicacionLider("RandomDeath")
					if resp == "death" {
						fmt.Println("Ha muerto")
						return
					}
					break
				}
			}
			ReadyToPlay = ""

			fmt.Println("Bienvenido al segundo juego: Tirar la cuerda")
			fmt.Println("Debera elegir un numero entre el 1 y el 4")
			fmt.Println("Si usted elige un numero de diferente paridad que el lider, quedara descalificado")
			fmt.Println("Esperando la eleccion del Lider")
			for {
				if ReadyToPlay == "Ready" {
					break
				}
			}
			ReadyToPlay = ""

			var num string
			fmt.Println("Elija un numero")
			fmt.Scanf("%s", &num)
			resp := ComunicacionLider(num)

			fmt.Println("Esperando al resultado ...")

			if resp == "wait" {
				for {
					if ReadyToPlay == "Ready" {
						resp := ComunicacionLider("resultado")
						if resp == "death" {
							fmt.Println("Ha muerto")
							return
						}
						fmt.Println("Gano en Tirar la cuerda")
						break
					}
				}
			}

			fmt.Println("El valor del pozo actual es: ", ComunicacionLider("ValPozo"))
			ReadyToPlay = ""
			JuegoActual = ""
		}

		if JuegoActual == "G3" {

			for {
				if ReadyToPlay == "Ready" {
					resp := ComunicacionLider("RandomDeath")
					if resp == "death" {
						fmt.Println("Ha muerto")
						return
					}
					break
				}
			}
			ReadyToPlay = ""

			fmt.Println("Bienvenido al tercer juego: Todo o nada")
			fmt.Println("Debera elegir un numero entre el 1 y el 10")
			fmt.Println("Si usted elige un numero muy lejano al del lider, quedara descalificado")
			fmt.Println("Esperando al Lider")
			for {
				if ReadyToPlay == "Ready" {
					break
				}
			}

			ReadyToPlay = ""
			var num string
			fmt.Println("Elija un numero")
			fmt.Scanf("%s", &num)
			resp := ComunicacionLider(num)

			fmt.Println("Esperando al resultado ...")

			if resp == "wait" {
				for {
					if ReadyToPlay == "Ready" {
						resp := ComunicacionLider("resultado")
						if resp == "death" {
							fmt.Println("Ha muerto")
							return
						}
						fmt.Println("Gano en Todo o Nada")
						break
					}
				}
			}
			fmt.Println("El valor del pozo actual es: ", ComunicacionLider("ValPozo"))
			ReadyToPlay = ""
			JuegoActual = ""
		}

	}

	<-forever
}
