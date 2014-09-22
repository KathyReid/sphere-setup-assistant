package main

import (
	"github.com/paypal/gatt"
	srplib "github.com/theojulienne/go-pkgs/crypto/srp"
	"crypto/sha256"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

const (
	StateAwaitingIntent = 0
	StateAwaitingBytesA = 1
	StateAwaitingBytesM = 2
	StateClientVerfied  = 3
)

type AuthHandler interface {
    GetUsername() string
    GetPassword() string
    AuthenticationInvalidated()
}

func RegisterSecuredRPCService(srv *gatt.Server, rpc_router JSONRPCRouter, auth_handler AuthHandler) {
	svc := srv.AddService(gatt.MustParseUUID(WifiConnnectionService))

	state := StateAwaitingIntent
	var ss *srplib.ServerSession = nil
	var skey []byte = nil
	var salt []byte = nil
	var cauth []byte = nil
	const ExpectedHashSizeBytes = (256/8)
	var secret_key []byte = nil
	const FirstResponseIV = 0x8000000000000000
	var last_enc_iv uint64 = 0
	var last_dec_iv uint64 = FirstResponseIV
	const RPCQueueSize = 32
	rpc_queue := make(chan []byte, RPCQueueSize)

	srp, err := srplib.NewSRP("rfc5054.2048", sha256.New, nil)
	if err != nil {
	    panic(err)
	}

	resetState := func() {
		state = StateAwaitingIntent
		ss = nil
		skey = nil
		salt = nil
		cauth = nil
		secret_key = nil
		last_enc_iv = 0
		last_dec_iv = FirstResponseIV
		rpc_queue = make(chan []byte, RPCQueueSize)

		auth_handler.AuthenticationInvalidated()

		// RFC2945 says:
		// x = SHA(<salt> | SHA(<username> | ":" | <raw password>))
		// whereas go srp does:
		// x = SHA(<salt> | <provided password>)
		// so we do the second SHA here:
		h := sha256.New()
		h.Write([]byte(auth_handler.GetUsername()))
		h.Write([]byte(":"))
		h.Write([]byte(auth_handler.GetPassword()))
		hashed_password := h.Sum(nil)

		salt_, verifier, err := srp.ComputeVerifier([]byte(hashed_password))
		if err != nil {
	 		panic(err)
		}
		salt = salt_
		ss = srp.NewServerSession([]byte(auth_handler.GetUsername()), salt, verifier)
	}

	resetState()

	svc.AddCharacteristic(gatt.MustParseUUID(ColorizeDisplay)).HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			if (state != StateAwaitingIntent) {
				return gatt.StatusUnexpectedError
			}

			log.Println("Pretend color: ", string(data))

			return gatt.StatusSuccess
		})

	svc.AddCharacteristic(gatt.MustParseUUID(PairIntentChar)).HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			if (state != StateAwaitingIntent && len(data) == 1 && data[0] == 0x01) {
				resetState();
				// reset and start!
			} else if (state != StateAwaitingIntent || len(data) != 1 || data[0] != 0x01) {
				resetState()
				return gatt.StatusUnexpectedError
			}

			state = StateAwaitingBytesA
			log.Println("State -> BytesA")

			return gatt.StatusSuccess
		})

	bac := svc.AddCharacteristic(gatt.MustParseUUID(SRPBytesAChar))
	MultiWritableCharacteristic(bac, 512, func(data []byte) byte {
			if (state != StateAwaitingBytesA) {
				resetState()
				return gatt.StatusUnexpectedError
			}

			log.Printf("Final data %d bytes: %v\n", len(data), data)
			skey_, err := ss.ComputeKey(data)
			if err != nil {
			    log.Fatal(err)
			    resetState()
				return gatt.StatusUnexpectedError
			}
			skey = skey_
			log.Printf("The Server's computed session key is %v len: %v\n", len(skey), skey)
			secret_key = skey

			state = StateAwaitingBytesM
			log.Println("State -> BytesM")

			return gatt.StatusSuccess
		})

	svc.AddCharacteristic(gatt.MustParseUUID(SRPBytesSChar)).HandleRead(
		gatt.ReadHandlerFunc(
			func(resp gatt.ReadResponseWriter, req *gatt.ReadRequest) {
				if (state != StateAwaitingBytesM) {
					resetState()
					resp.SetStatus(gatt.StatusUnexpectedError)
					return
				}

				log.Println("Sent BytesS:", salt)
				ChunkWrite(req, resp, salt)

				return
			}))

	svc.AddCharacteristic(gatt.MustParseUUID(SRPBytesBChar)).HandleRead(
		gatt.ReadHandlerFunc(
			func(resp gatt.ReadResponseWriter, req *gatt.ReadRequest) {
				if (state != StateAwaitingBytesM) {
					resetState()
					resp.SetStatus(gatt.StatusUnexpectedError)
					return
				}

				log.Println("Send BytesB:", ss.GetB())
				ChunkWrite(req, resp, ss.GetB())

				return
			}))

	bmc := svc.AddCharacteristic(gatt.MustParseUUID(SRPBytesMChar))
	MultiWritableCharacteristic(bmc, 256, func(data []byte) byte {
			if (state != StateAwaitingBytesM || len(data) != ExpectedHashSizeBytes) {
				log.Println("Received bytes", len(data))
				resetState()
				return gatt.StatusUnexpectedError
			}

			if !ss.VerifyClientAuthenticator(data) {
			    log.Println("Client Authenticator is not valid")
			    resetState()
				return gatt.StatusUnexpectedError
			}

			cauth = data
			state = StateClientVerfied
			last_enc_iv = 0
			last_dec_iv = FirstResponseIV
			log.Println("State -> StateClientVerfied")

			return gatt.StatusSuccess
		})

	svc.AddCharacteristic(gatt.MustParseUUID(SRPBytesHAMKChar)).HandleRead(
		gatt.ReadHandlerFunc(
			func(resp gatt.ReadResponseWriter, req *gatt.ReadRequest) {
				if (state != StateClientVerfied) {
					resetState()
					resp.SetStatus(gatt.StatusUnexpectedError)
					return
				}

				ChunkWrite(req, resp, ss.ComputeAuthenticator(cauth))

				log.Println("Send Authenticator")
			}))

	rpc := svc.AddCharacteristic(gatt.MustParseUUID(CommsChanChar))
	MultiWritableCharacteristic(rpc, 1024, func(data []byte) byte {
		if (state != StateClientVerfied) {
			resetState()
			return gatt.StatusUnexpectedError
		}

		const TransportIVSize = 8

		t_enc_iv := binary.LittleEndian.Uint64(data[:TransportIVSize])
		data = data[TransportIVSize:]

		if t_enc_iv <= last_enc_iv || t_enc_iv >= FirstResponseIV {
			// not allowed to re-use IVs. must be strictly increasing
			return gatt.StatusUnexpectedError
		}

		last_enc_iv = t_enc_iv // mark as used

		log.Println("Received encrypted data", data)
		rpc_in, err := decrypt(secret_key, data, t_enc_iv)
		if err != nil {
			fmt.Println("decrypt failed:", err)
			return gatt.StatusUnexpectedError
		}

		log.Println("Received data", rpc_in)
		resp_channel := rpc_router.CallRaw(bytes.Trim(rpc_in, "\x00"))

		// make the response here, at any time!
		go func() {
			//response_raw := []byte(rpc_in)
			response_raw := <- resp_channel

			last_dec_iv += 1
			rpc_out, err := encrypt(secret_key, response_raw, last_dec_iv)
			if err != nil {
				fmt.Println("encrypt failed:", err)
				return
			}

			tmp_response := make([]byte, len(rpc_out) + TransportIVSize)
			binary.LittleEndian.PutUint64(tmp_response, last_dec_iv)
			copy(tmp_response[TransportIVSize:], rpc_out)

			rpc_queue <- tmp_response
		}()

		return gatt.StatusSuccess
	})
	rpc.HandleNotifyFunc(
		func(r gatt.Request, n gatt.Notifier) {
			go func() {
				for !n.Done() {
					full_msg := <- rpc_queue
					fmt.Printf("Ready to send: %v\n", full_msg)

					const SizePerMessage = 16

					for i := 0; i < len(full_msg); i+=SizePerMessage {
						flags := 0
						// mark final message
						if i+SizePerMessage >= len(full_msg) {
							flags |= 0x8000
						}

						end := i + SizePerMessage
						if end > len(full_msg) {
							end = len(full_msg)
						}

						to_send := full_msg[i:end]
						buffer := make([]byte, len(to_send) + 2)
						binary.LittleEndian.PutUint16(buffer, uint16(i | flags))
						copy(buffer[2:], to_send)

						n.Write(buffer)
						fmt.Printf("Sending data: %v\n", buffer)
					}
				}
			}()
		})
}
