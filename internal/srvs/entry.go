/* Copyright (c) 2018 FurtherSystem Co.,Ltd. All rights reserved.

   This program is free software; you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation; version 2 of the License.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program; if not, write to the Free Software
   Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1335  USA */

package srvs

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"openrelay/internal/defs"
	"strconv"
	"strings"
	"time"
)

func (o *OpenRelay) EntryServ() {
	http.HandleFunc("/version", version)
	http.HandleFunc("/logon", logon)
	http.HandleFunc("/rooms", o.Rooms)
	http.HandleFunc("/room/info", o.roomInfo)
	http.HandleFunc("/room/create/", o.Create)
	http.HandleFunc("/room/join_prepare_polling/", o.JoinPreparePolling)
	http.HandleFunc("/room/join_prepare_complete/", o.JoinPrepareComplete)
	http.HandleFunc("/room/prop/", o.RoomProp)
	http.HandleFunc("/logoff", logoff)
	s := &http.Server{
		Addr:              o.EntryHost + ":" + o.EntryPort,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       10 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}

func version(w http.ResponseWriter, r *http.Request) {
	if !validateGet(w, r) {
		return
	}
	log.Println(defs.VERBOSE, "version called.")

	switch r.Header.Get("User-Agent") {
	case defs.UA_UNITY_CDK:
		log.Println(defs.VVERBOSE, "UA_UNITY_CDK")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(defs.REQUIRE_UNITY_CDK_VERSION))
	case defs.UA_UE4_CDK:
		log.Println(defs.VVERBOSE, "UA_UE4_CDK")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(defs.REQUIRE_UE4_CDK_VERSION))
	case defs.UA_NATIVE_CDK:
		log.Println(defs.VVERBOSE, "UA_NATIVE_CDK")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(defs.REQUIRE_NATIVE_CDK_VERSION))
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func logon(w http.ResponseWriter, r *http.Request) {
	validatePost(w, r)
	log.Println(defs.VERBOSE, "logon called.")
	w.Write([]byte("OK")) //TODO sessid
}

func addLogonResponse(relay *defs.RoomInstance) ([]byte, error) {
	log.Println(defs.VVERBOSE, "addLogonResponse called.")
	return nil, nil
}

func (o *OpenRelay) Rooms(w http.ResponseWriter, r *http.Request) {
	validateGet(w, r)
	log.Println(defs.VERBOSE, "Rooms called.")

	var err error
	writeBuf := new(bytes.Buffer)
	if 0 < len(o.ReserveRooms) {
		writeBuf, err = o.addResponseBytes(writeBuf, defs.OPENRELAY_RESPONSE_CODE_OK)
		err = binary.Write(writeBuf, binary.LittleEndian, uint16(len(o.ReserveRooms)))
		if err != nil {
			log.Error("binary write failed. ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_RESPONSE_WRITE_FAILED))
			return
		}
		for _, roomId := range o.ReserveRooms {
			roomIdHexStr := defs.GuidFormatString(roomId)
			writeBuf, err = o.addRoomResponse(writeBuf, *o.RelayQueue[roomIdHexStr], *o.RoomQueue[roomIdHexStr])
			if err != nil {
				log.Error("binary write failed. ", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_RESPONSE_WRITE_FAILED))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(writeBuf.Bytes())
	} else {
		writeBuf, err = o.addResponseBytes(writeBuf, defs.OPENRELAY_RESPONSE_CODE_OK_NO_ROOM)
		binary.Write(writeBuf, binary.LittleEndian, uint16(0)) // alignment
		w.WriteHeader(http.StatusOK)
		w.Write(writeBuf.Bytes())
	}
}

func (o *OpenRelay) roomsResponse(relay *defs.RoomInstance) ([]byte, error) {
	log.Println(defs.VVERBOSE, "roomsResponse called.")
	return nil, nil
}

func (o *OpenRelay) roomInfo(w http.ResponseWriter, r *http.Request) {
	validateGet(w, r)
	log.Println(defs.VERBOSE, "roomsInfo called.")
	w.Write([]byte("OK"))
}

func (o *OpenRelay) roomInfoResponse(relay *defs.RoomInstance) ([]byte, error) {
	log.Println(defs.VVERBOSE, "roomsInfoResponse called.")
	return nil, nil
}

func (o *OpenRelay) Create(w http.ResponseWriter, r *http.Request) {
	validatePost(w, r)
	log.Println(defs.VERBOSE, "Create called.")
	if len(o.HotRoomQueue) <= 0 {
		log.Println(defs.NOTICE, "room capacity over.")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_CREATE_ROOM_CAPACITY_OVER))
		return
	}
	requestName := strings.Replace(r.URL.Path, "/room/create/", "", 1)
	_, exist := o.ReserveRooms[requestName]
	var err error
	var roomId [16]byte
	var roomIdHexStr string
	writeBuf := new(bytes.Buffer)
	if exist {
		roomId = o.ReserveRooms[requestName]
		roomIdHexStr = defs.GuidFormatString(roomId)
		writeBuf, err = o.addResponseBytes(writeBuf, defs.OPENRELAY_RESPONSE_CODE_NG_CREATE_ROOM_ALREADY_EXISTS)
		if err != nil {
			log.Error("binary write failed. ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_RESPONSE_WRITE_FAILED))
			return
		}
		binary.Write(writeBuf, binary.LittleEndian, uint16(0)) // alignment
	} else {
		roomId = o.HotRoomQueue[0]
		roomIdHexStr = defs.GuidFormatString(roomId)
		o.HotRoomQueue = o.HotRoomQueue[1:]
		// reserve immediately
		o.ReserveRooms[requestName] = roomId
		o.ResolveRoomIds[roomIdHexStr] = requestName
		body := make([]byte, 2) //uint16 size
		_, err := r.Body.Read(body)
		if err != nil && err != io.EOF {
			log.Error("polling failed. ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_REQUEST_READ_FAILED))
			return
		}
		readBuf := bytes.NewReader(body)

		var maxPlayers uint16
		err = binary.Read(readBuf, binary.LittleEndian, &maxPlayers)
		if err != nil {
			log.Error("binary read failed. invalid request data", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_REQUEST_READ_FAILED))
			return
		}
		o.RoomQueue[roomIdHexStr].Name = requestName
		o.RoomQueue[roomIdHexStr].Filter = ""
		o.RoomQueue[roomIdHexStr].Capacity = maxPlayers

		writeBuf, err = o.addResponseBytes(writeBuf, defs.OPENRELAY_RESPONSE_CODE_OK_ROOM_ASSGIN_AND_CREATED)
		if err != nil {
			log.Error("binary write failed. ", err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_RESPONSE_WRITE_FAILED))
			return
		}
		binary.Write(writeBuf, binary.LittleEndian, uint16(0)) // alignment
	}

	writeBuf, err = o.addRoomResponse(writeBuf, *o.RelayQueue[roomIdHexStr], *o.RoomQueue[roomIdHexStr])
	if err != nil {
		log.Error("binary write failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_RESPONSE_WRITE_FAILED))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(writeBuf.Bytes())
}

func (o *OpenRelay) getResponseBytes(code defs.ResponseCode) []byte {
	log.Println(defs.VVERBOSE, "getResponseBytes called.")
	writeBuf := new(bytes.Buffer)
	binary.Write(writeBuf, binary.LittleEndian, code)      //ignore error ok.
	binary.Write(writeBuf, binary.LittleEndian, uint16(0)) //ignore error ok.
	return writeBuf.Bytes()
}

func (o *OpenRelay) addResponseBytes(writeBuf *bytes.Buffer, code defs.ResponseCode) (*bytes.Buffer, error) {
	log.Println(defs.VVERBOSE, "addResponseBytes called.")
	var err error
	err = binary.Write(writeBuf, binary.LittleEndian, code)
	if err != nil {
		return nil, err
	}
	return writeBuf, nil
}

func (o *OpenRelay) addRoomResponse(writeBuf *bytes.Buffer, relay defs.RoomInstance, room defs.RoomParameter) (*bytes.Buffer, error) {
	log.Println(defs.VVERBOSE, "addRoomResponse called.")
	var err error
	roomRes := defs.RoomResponse{}
	roomRes.Id = room.Id
	roomRes.Capacity = room.Capacity
	roomRes.UserCount = uint16(len(relay.Guids))
	roomRes.QueuingPolicy = room.QueuingPolicy
	roomRes.Flags = 0 ^ 7 | 0 ^ 6 | 0 ^ 5 | 0 ^ 4 | 0 ^ 3 | 0 ^ 2 | 0 ^ 1 | 0
	roomRes.StfDealPort = room.StfDealPort
	roomRes.StfSubPort = room.StfSubPort
	roomRes.StlDealPort = room.StlDealPort
	roomRes.StlSubPort = room.StlSubPort
	roomRes.NameLen = byte(len(room.Name))
	roomRes.FilterLen = byte(len(room.Filter))
	if 0 < roomRes.NameLen {
		roomRes.Name = [256]byte{}
		copy(roomRes.Name[:roomRes.NameLen], room.Name[:roomRes.NameLen])
	}
	if 0 < roomRes.FilterLen {
		roomRes.Filter = [256]byte{}
		copy(roomRes.Filter[:roomRes.FilterLen], room.Filter[:roomRes.FilterLen])
	}
	roomRes.ListenMode = byte(o.ListenMode)
	ipv4Addr, err := net.ResolveIPAddr("ip4", o.ListenIpv4)
	if err != nil {
		return nil, err
	}
	copy(roomRes.ListenAddrIpv4[:], ipv4Addr.IP.To4()[:4])
	ipv6Addr, err := net.ResolveIPAddr("ip6", o.ListenIpv6)
	if err != nil {
		return nil, err
	}
	copy(roomRes.ListenAddrIpv6[:], ipv6Addr.IP.To16())
	err = binary.Write(writeBuf, binary.LittleEndian, roomRes)
	if err != nil {
		return nil, err
	}

	log.Printf(defs.INFO, "response room roomId :%s", defs.GuidFormatString(roomRes.Id))

	log.Printf(defs.VERBOSE, "response room max players: %d", int(roomRes.Capacity))
	log.Printf(defs.VERBOSE, "response room UserCount :%d", roomRes.UserCount)
	log.Printf(defs.VERBOSE, "response room statefull deal port: %d", roomRes.StfDealPort)
	log.Printf(defs.VERBOSE, "response room statefull subscribe port: %d", roomRes.StfSubPort)
	log.Printf(defs.VERBOSE, "response room stateless deal port: %d", roomRes.StlDealPort)
	log.Printf(defs.VERBOSE, "response room stateless subscribe port: %d", roomRes.StlSubPort)
	log.Printf(defs.VERBOSE, "response room name :%s", roomRes.Name[:roomRes.NameLen])
	log.Printf(defs.VERBOSE, "response room name length :%d", roomRes.NameLen)
	log.Printf(defs.VERBOSE, "response room filter :%s", roomRes.Filter[:roomRes.FilterLen])
	log.Printf(defs.VERBOSE, "response room filter length :%d", roomRes.FilterLen)
	log.Printf(defs.VERBOSE, "response room listen mode :%d", roomRes.ListenMode)
	log.Printf(defs.VERBOSE, "response room listen addr ipv4(origin) :%s", o.ListenIpv4)
	log.Printf(defs.VERBOSE, "response room listen addr ipv4(resolve addr) :%s", ipv4Addr.IP.String())
	log.Printf(defs.VERBOSE, "response room listen addr ipv4(parsed) :%x", roomRes.ListenAddrIpv4)
	log.Printf(defs.VERBOSE, "response room listen addr ipv6(origin) :%s", o.ListenIpv6)
	log.Printf(defs.VERBOSE, "response room listen addr ipv6(resolve addr) :%s", ipv6Addr.IP.String())
	log.Printf(defs.VERBOSE, "response room listen addr ipv6(parsed) :%x", roomRes.ListenAddrIpv6)
	return writeBuf, nil
}

func (o *OpenRelay) JoinPreparePolling(w http.ResponseWriter, r *http.Request) {
	validatePut(w, r)
	log.Println(defs.VERBOSE, "JoinPreparePolling called.")
	requestName := strings.Replace(r.URL.Path, "/room/join_prepare_polling/", "", 1)
	roomId, exist := o.ReserveRooms[requestName]
	if !exist {
		log.Println(defs.NOTICE, "room not found.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	roomIdHexStr := defs.GuidFormatString(roomId)
	room, _ := o.RoomQueue[roomIdHexStr]
	relay, _ := o.RelayQueue[roomIdHexStr]
	joinPollingQueue := o.JoinAllPollingQueue[roomIdHexStr]
	joinProcessQueue := o.JoinAllProcessQueue[roomIdHexStr]
	joinTimeoutQueue := o.JoinAllTimeoutQueue[roomIdHexStr]
	joinProcessQueueLen := 0
	if joinProcessQueue.Seed != "" {
		joinProcessQueueLen = 1
	}
	if len(relay.Uids) >= int(room.Capacity) && room.QueuingPolicy == defs.BLOCK_ROOM_MAX {
		log.Println(defs.VERBOSE, "OK request name: %s, roomId: %s, capacity: %d",  requestName, roomIdHexStr, int(room.Capacity))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("OK " + requestName + " " + roomIdHexStr + " " + strconv.Itoa(int(room.Capacity))))
		return
	} else if len(relay.Uids)+joinProcessQueueLen+len(joinPollingQueue) >= int(room.Capacity) && room.QueuingPolicy == defs.BLOCK_ROOM_AND_QUEUE_MAX {
		log.Println(defs.VERBOSE, "OK request name: %s, roomId: %s, capacity: %d",  requestName, roomIdHexStr, int(room.Capacity))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("OK"))
		return
	}

	length, err := strconv.Atoi(r.Header.Get("Content-Length"))
	if err != nil {
		log.Error("polling failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body := make([]byte, length)
	length, err = r.Body.Read(body)
	if err != nil && err != io.EOF {
		log.Error("polling failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	readBuf := bytes.NewReader(body[:length])

	joinSeed, err := o.readJoinSeed(readBuf)
	if err != nil {
		log.Error("polling failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	hexJoinSeed := hex.EncodeToString(joinSeed)

	if joinProcessQueue.Timestamp+int64(o.JoinTimeout) < time.Now().Unix() {
		o.JoinAllProcessQueue[roomIdHexStr] = defs.RoomJoinRequest{Seed: "", Timestamp: 0}
		o.JoinAllTimeoutQueue[roomIdHexStr] = append(o.JoinAllTimeoutQueue[roomIdHexStr], joinProcessQueue)
	}
	if len(joinTimeoutQueue) > 0 {
		var needTimeoutResponse bool
		for _, request := range joinTimeoutQueue {
			if request.Seed == hexJoinSeed {
				needTimeoutResponse = true
			}
		}
		joinTimeoutQueue := make([]defs.RoomJoinRequest, 0)
		o.JoinAllTimeoutQueue[roomIdHexStr] = joinTimeoutQueue
		if needTimeoutResponse {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}
	}

	if joinProcessQueue.Seed == "" {
		if len(joinPollingQueue) == 0 {
			res, err := o.JoinPrepareResponse(relay, joinSeed)
			if err != nil {
				log.Println(defs.NOTICE, "polling failed. ", err)
				w.WriteHeader(http.StatusBadRequest)
			} else {
				joinProcessQueue.Seed = hexJoinSeed
				joinProcessQueue.Timestamp = time.Now().Unix()
				o.JoinAllProcessQueue[roomIdHexStr] = joinProcessQueue
				w.WriteHeader(http.StatusOK)
				w.Write(res)
			}
			return
		} else if check := hex.EncodeToString(joinPollingQueue[0]); check == hexJoinSeed {
			res, err := o.JoinPrepareResponse(relay, joinSeed)
			if err != nil {
				log.Println(defs.NOTICE, "polling failed. ", err)
				w.WriteHeader(http.StatusBadRequest)
			} else {
				joinProcessQueue.Seed = hexJoinSeed
				joinProcessQueue.Timestamp = time.Now().Unix()
				o.JoinAllProcessQueue[roomIdHexStr] = joinProcessQueue
				joinPollingQueue = joinPollingQueue[1:] //pop
				o.JoinAllPollingQueue[roomIdHexStr] = joinPollingQueue
				w.WriteHeader(http.StatusOK)
				w.Write(res)
			}
			return
		} else {
			if !contains(joinPollingQueue, joinSeed) {
				joinPollingQueue = append(joinPollingQueue, joinSeed)
				o.JoinAllPollingQueue[roomIdHexStr] = joinPollingQueue
			}
			w.WriteHeader(http.StatusContinue)
			return
		}

	} else {
		if !contains(joinPollingQueue, joinSeed) {
			joinPollingQueue = append(joinPollingQueue, joinSeed)
			o.JoinAllPollingQueue[roomIdHexStr] = joinPollingQueue
		}
		w.WriteHeader(http.StatusContinue)
		return
	}
}

func contains(slice [][]byte, elem []byte) bool {
	for _, value := range slice {
		if hex.EncodeToString(elem) == hex.EncodeToString(value) {
			return true
		}
	}
	return false
}

//func readHeader(readBuf *bytes.Reader, header Header) (Header, error) {
//	err := binary.Read(readBuf, binary.LittleEndian, &header)
//	if err != nil {
//		return header, err
//	}
//	if header.Ver != FrameVersion {
//		return header, fmt.Errorf("invalid FrameVersion %d != %d", FrameVersion, header.Ver)
//	}
//
//	log.Printf(defs.VVERBOSE, "received header.Ver: '%d' ", header.Ver)
//	log.Printf(defs.VVERBOSE, "received header.RelayCode: '%d' ", header.RelayCode)
//	log.Printf(defs.VVERBOSE, "received header.ContentCode: '%d' ", header.ContentCode)
//	log.Printf(defs.VVERBOSE, "received header.DestCode: '%d' ", header.DestCode)
//	log.Printf(defs.VVERBOSE, "received header.Mask: '%d' ", header.Mask)
//	log.Printf(defs.VVERBOSE, "received header.SrcUid: '%d' ", header.SrcUid)
//	log.Printf(defs.VVERBOSE, "received header.SrcOid: '%d' ", header.SrcOid)
//	log.Printf(defs.VVERBOSE, "received header.DestLen: '%d' ", header.DestLen)
//	log.Printf(defs.VVERBOSE, "received header.ContentLen: '%d' ", header.ContentLen)
//	return header, nil
//}

func (o *OpenRelay) readJoinSeed(readBuf *bytes.Reader) ([]byte, error) {
	var seedLen uint16
	err := binary.Read(readBuf, binary.LittleEndian, &seedLen)
	if err != nil {
		return nil, err
	}

	log.Printf(defs.VVERBOSE, "received join seedLen: '%d' ", seedLen)

	joinSeed := make([]byte, seedLen)
	err = binary.Read(readBuf, binary.LittleEndian, &joinSeed)
	if err != nil {
		return nil, err
	}

	log.Printf(defs.VVERBOSE, "received join seed: '%s' ", hex.EncodeToString(joinSeed))
	return joinSeed, nil
}

func (o *OpenRelay) JoinPrepareResponse(relay *defs.RoomInstance, joinSeed []byte) ([]byte, error) {
	log.Println(defs.VERBOSE, "JoinPrepareResponse called.")
	var err error
	writeBuf := new(bytes.Buffer)
	relay.LastUid += 1
	if relay.MasterUidNeed {
		relay.MasterUidNeed = false
		relay.MasterUid = relay.LastUid
	}
	joinedUids := []defs.PlayerId{}
	for k, _ := range relay.Uids {
		joinedUids = append(joinedUids, k)
	}
	assginUid := relay.LastUid
	relay.Guids[string(joinSeed)] = relay.LastUid
	relay.Uids[relay.LastUid] = string(joinSeed)
	joinedUidsLen := uint16(len(joinedUids))
	joinedNamesLen := uint16(len(relay.Names))
	alignmentLen := uint16(0)
	alignment := []byte{}
	relay.Hbs[assginUid] = time.Now().Unix()
	log.Println(defs.INFO, "-> join ", relay.LastUid, ", seed ", hex.EncodeToString(joinSeed))

	err = binary.Write(writeBuf, binary.LittleEndian, relay.MasterUid)
	if err != nil {
		return nil, err
	}
	err = binary.Write(writeBuf, binary.LittleEndian, assginUid)
	if err != nil {
		return nil, err
	}
	err = binary.Write(writeBuf, binary.LittleEndian, joinedUidsLen)
	if err != nil {
		return nil, err
	}

	err = binary.Write(writeBuf, binary.LittleEndian, joinedNamesLen)
	if err != nil {
		return nil, err
	}

	err = binary.Write(writeBuf, binary.LittleEndian, joinedUids)
	if err != nil {
		return nil, err
	}
	//write adjust alignment at joinedUidsLen.
	alignmentLen = joinedUidsLen % 4
	if alignmentLen != 0 {
		alignment = make([]byte, alignmentLen)
		err = binary.Write(writeBuf, binary.LittleEndian, alignment)
		if err != nil {
			return nil, err
		}
	}

	for _, name := range relay.Names {
		nameBytes := []byte(name)
		nameLen := uint16(len(name))

		err = binary.Write(writeBuf, binary.LittleEndian, nameLen)
		if err != nil {
			return nil, err
		}

		err = binary.Write(writeBuf, binary.LittleEndian, nameBytes)
		if err != nil {
			return nil, err
		}
		//write adjust alignment at nameLen.
		alignmentLen = (2 + nameLen) % 4
		if alignmentLen != 0 {
			alignment = make([]byte, alignmentLen)
			err = binary.Write(writeBuf, binary.LittleEndian, alignment)
			if err != nil {
				return nil, err
			}
		}
	}
	log.Println(defs.VERBOSE, "JoinPrepareResponse finished.")
	return writeBuf.Bytes(), nil
}

func (o *OpenRelay) RoomProp(w http.ResponseWriter, r *http.Request) {
	validateGet(w, r)
	log.Println(defs.VERBOSE, "RoomProp called.")
	requestName := strings.Replace(r.URL.Path, "/room/prop/", "", 1)
	var err error
	roomId, _ := o.ReserveRooms[requestName]
	roomIdHexStr := defs.GuidFormatString(roomId)
	relay, _ := o.RelayQueue[roomIdHexStr]
	contentLen := uint16(len(relay.Props[defs.PropKeyLegacy]))
	properties := relay.Props[defs.PropKeyLegacy]

	writeBuf := new(bytes.Buffer)
	writeBuf, err = o.addResponseBytes(writeBuf, defs.OPENRELAY_RESPONSE_CODE_OK)
	if err != nil {
		log.Error("binary write failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_RESPONSE_WRITE_FAILED))
		return
	}
	err = binary.Write(writeBuf, binary.LittleEndian, contentLen)
	if err != nil {
		log.Error("binary write failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_RESPONSE_WRITE_FAILED))
		return
	}

	err = binary.Write(writeBuf, binary.LittleEndian, properties)
	if err != nil {
		log.Error("binary write failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(o.getResponseBytes(defs.OPENRELAY_RESPONSE_CODE_NG_RESPONSE_WRITE_FAILED))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(writeBuf.Bytes())
}

func (o *OpenRelay) JoinPrepareComplete(w http.ResponseWriter, r *http.Request) {
	validatePost(w, r)
	log.Println(defs.VERBOSE, "JoinPrepareComplete called.")
	requestName := strings.Replace(r.URL.Path, "/room/join_prepare_complete/", "", 1)
	roomId, exist := o.ReserveRooms[requestName]
	if !exist {
		log.Println(defs.NOTICE, "room not found.")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	length, err := strconv.Atoi(r.Header.Get("Content-Length"))
	if err != nil {
		log.Error("polling failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body := make([]byte, length)
	length, err = r.Body.Read(body)
	if err != nil && err != io.EOF {
		log.Error("polling failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	readBuf := bytes.NewReader(body[:length])

	joinSeed, err := o.readJoinSeed(readBuf)
	if err != nil {
		log.Error("polling failed. ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	roomIdHexStr := defs.GuidFormatString(roomId)
	joinProcessQueue := o.JoinAllProcessQueue[roomIdHexStr]
	hexJoinSeed := hex.EncodeToString(joinSeed)
	if joinProcessQueue.Seed == hexJoinSeed {
		log.Printf(defs.INFO, "seed is match %s == %s \n", joinProcessQueue.Seed, hexJoinSeed)
		joinProcessQueue := defs.RoomJoinRequest{Seed: "", Timestamp: 0}
		o.JoinAllProcessQueue[roomIdHexStr] = joinProcessQueue
		w.WriteHeader(http.StatusOK)
		return
	} else {
		log.Printf(defs.NOTICE, "seed is not match %s != %s \n", joinProcessQueue.Seed, hexJoinSeed)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func logoff(w http.ResponseWriter, r *http.Request) {
	validatePost(w, r)
	log.Println(defs.VERBOSE, "logoff called.")
	w.Write([]byte("OK"))
}

func validateGet(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return false
	}
	return true
}

func validatePost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return false
	}
	return true
}

func validatePut(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusNotFound)
		return false
	}
	return true
}
