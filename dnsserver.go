package main

/*
	Simple DNS Server implemented in Go for humandns

	BSD 2-Clause License

	Copyright (c) 2019, Daniel Lorch
	Copyright (c) 2023, Haoxi Tan

	All rights reserved.

	Redistribution and use in source and binary forms, with or without
	modification, are permitted provided that the following conditions are met:

	1. Redistributions of source code must retain the above copyright notice, this
	   list of conditions and the following disclaimer.

	2. Redistributions in binary form must reproduce the above copyright notice,
       this list of conditions and the following disclaimer in the documentation
       and/or other materials provided with the distribution.

	THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
	AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
	IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
	DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
	FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
	DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
	SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
	CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
	OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
	OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

import (
	"bytes"
	"encoding/binary"
	// "fmt"
	"log"
	"net"
	"os"
	"flag"
	"strings"
	"github.com/go-redis/redis"
)

var redisClient *redis.Client 
var ExpiryTimeInSeconds uint

// DNSHeader describes the request/response DNS header
type DNSHeader struct {
	TransactionID  uint16
	Flags          uint16
	NumQuestions   uint16
	NumAnswers     uint16
	NumAuthorities uint16
	NumAdditionals uint16
}

// DNSResourceRecord describes individual records in the request and response of the DNS payload body
type DNSResourceRecord struct {
	DomainName         string
	Type               uint16
	Class              uint16
	TimeToLive         uint32
	ResourceDataLength uint16
	ResourceData       []byte
}

// Type and Class values for DNSResourceRecord
const (
	TypeA                  uint16 = 1 // a host address
	TypeAAAA			   uint16 = 28 // ipv6 addr
	ClassINET              uint16 = 1 // the Internet
	FlagResponse           uint16 = 1 << 15
	UDPMaxMessageSizeBytes uint   = 512 // RFC1035
)

// Look up values in a database
func dbLookup(queryResourceRecord DNSResourceRecord) ([]DNSResourceRecord, []DNSResourceRecord, []DNSResourceRecord) {
	var answerResourceRecords = make([]DNSResourceRecord, 0)
	var authorityResourceRecords = make([]DNSResourceRecord, 0)
	var additionalResourceRecords = make([]DNSResourceRecord, 0)

	// TODO add support for IPv6 lookup
	if queryResourceRecord.Class != ClassINET {
		return answerResourceRecords, authorityResourceRecords, additionalResourceRecords
	}

	if queryResourceRecord.Type == TypeA || queryResourceRecord.Type == TypeAAAA {
		//queryResourceRecord.DomainName
		resolvedAddress := redisClient.Get(queryResourceRecord.DomainName)
		if resolvedAddress.Val() == "" { // not in db, probably should return NXDOMAIN instead
			return answerResourceRecords, authorityResourceRecords, additionalResourceRecords
		}

		parsedAddress := net.ParseIP(resolvedAddress.Val())
		log.Printf("%s resolved to %s (parsed %#v)", queryResourceRecord.DomainName, resolvedAddress, parsedAddress)

		// if queryResourceRecord.Type == TypeA {
		if strings.Contains(queryResourceRecord.DomainName, "ip4") {
			if queryResourceRecord.Type == TypeA {
				answerResourceRecords = append(answerResourceRecords, DNSResourceRecord{
					DomainName:         queryResourceRecord.DomainName,
					Type:               TypeA,
					Class:              ClassINET,
					TimeToLive:         uint32(ExpiryTimeInSeconds),
					ResourceData:       parsedAddress[12:16], // ipv4 address
					ResourceDataLength: 4,
				})

			}
			

		} else if strings.Contains(queryResourceRecord.DomainName, "ip6") {

			if queryResourceRecord.Type == TypeAAAA  {
				answerResourceRecords = append(answerResourceRecords, DNSResourceRecord{
					DomainName:         queryResourceRecord.DomainName,
					Type:               TypeAAAA,
					Class:              ClassINET,
					TimeToLive:         uint32(ExpiryTimeInSeconds),
					ResourceData:       parsedAddress, // ipv6 address
					ResourceDataLength: 16,
				})
			} else { // if they queried a ipv6 name without querying the type AAAA, put it in the additional records
				additionalResourceRecords = append(additionalResourceRecords, DNSResourceRecord{
					DomainName:         queryResourceRecord.DomainName,
					Type:               TypeAAAA,
					Class:              ClassINET,
					TimeToLive:         uint32(ExpiryTimeInSeconds),
					ResourceData:       parsedAddress, // ipv6 address
					ResourceDataLength: 16,
				})
			}
			
		}

	}

	
	

	
	


	return answerResourceRecords, authorityResourceRecords, additionalResourceRecords
}

// RFC1035: "Domain names in messages are expressed in terms of a sequence
// of labels. Each label is represented as a one octet length field followed
// by that number of octets.  Since every domain name ends with the null label
// of the root, a domain name is terminated by a length byte of zero."
func readDomainName(requestBuffer *bytes.Buffer) (string, error) {
	var domainName string

	b, err := requestBuffer.ReadByte()

	for ; b != 0 && err == nil; b, err = requestBuffer.ReadByte() {
		labelLength := int(b)
		labelBytes := requestBuffer.Next(labelLength)
		labelName := string(labelBytes)

		if len(domainName) == 0 {
			domainName = labelName
		} else {
			domainName += "." + labelName
		}
	}

	return domainName, err
}

// RFC1035: "Domain names in messages are expressed in terms of a sequence
// of labels. Each label is represented as a one octet length field followed
// by that number of octets.  Since every domain name ends with the null label
// of the root, a domain name is terminated by a length byte of zero."
func writeDomainName(responseBuffer *bytes.Buffer, domainName string) error {
	labels := strings.Split(domainName, ".")

	for _, label := range labels {
		labelLength := len(label)
		labelBytes := []byte(label)

		responseBuffer.WriteByte(byte(labelLength))
		responseBuffer.Write(labelBytes)
	}

	err := responseBuffer.WriteByte(byte(0))

	return err
}

func handleDNSClient(requestBytes []byte, serverConn *net.UDPConn, clientAddr *net.UDPAddr) {
	/**
	 * read request
	 */
	var requestBuffer = bytes.NewBuffer(requestBytes)
	var queryHeader DNSHeader
	var queryResourceRecords []DNSResourceRecord

	err := binary.Read(requestBuffer, binary.BigEndian, &queryHeader) // network byte order is big endian

	if err != nil {
		log.Println("Error decoding header: ", err.Error())
	}

	queryResourceRecords = make([]DNSResourceRecord, queryHeader.NumQuestions)

	for idx, _ := range queryResourceRecords {
		queryResourceRecords[idx].DomainName, err = readDomainName(requestBuffer)

		if err != nil {
			log.Println("Error decoding label: ", err.Error())
		}

		queryResourceRecords[idx].Type = binary.BigEndian.Uint16(requestBuffer.Next(2))
		queryResourceRecords[idx].Class = binary.BigEndian.Uint16(requestBuffer.Next(2))
	}

	/**
	 * lookup values
	 */
	var answerResourceRecords = make([]DNSResourceRecord, 0)
	var authorityResourceRecords = make([]DNSResourceRecord, 0)
	var additionalResourceRecords = make([]DNSResourceRecord, 0)

	for _, queryResourceRecord := range queryResourceRecords {
		newAnswerRR, newAuthorityRR, newAdditionalRR := dbLookup(queryResourceRecord)

		answerResourceRecords = append(answerResourceRecords, newAnswerRR...) // three dots cause the two lists to be concatenated
		authorityResourceRecords = append(authorityResourceRecords, newAuthorityRR...)
		additionalResourceRecords = append(additionalResourceRecords, newAdditionalRR...)
	}

	/**
	 * write response
	 */
	var responseBuffer = new(bytes.Buffer)
	var responseHeader DNSHeader

	responseHeader = DNSHeader{
		TransactionID:  queryHeader.TransactionID,
		Flags:          FlagResponse,
		NumQuestions:   queryHeader.NumQuestions,
		NumAnswers:     uint16(len(answerResourceRecords)),
		NumAuthorities: uint16(len(authorityResourceRecords)),
		NumAdditionals: uint16(len(additionalResourceRecords)),
	}

	err = Write(responseBuffer, &responseHeader)

	if err != nil {
		log.Println("Error writing to buffer: ", err.Error())
	}

	for _, queryResourceRecord := range queryResourceRecords {
		err = writeDomainName(responseBuffer, queryResourceRecord.DomainName)

		if err != nil {
			log.Println("Error writing to buffer: ", err.Error())
		}

		Write(responseBuffer, queryResourceRecord.Type)
		Write(responseBuffer, queryResourceRecord.Class)
	}

	for _, answerResourceRecord := range answerResourceRecords {
		err = writeDomainName(responseBuffer, answerResourceRecord.DomainName)

		if err != nil {
			log.Println("Error writing to buffer: ", err.Error())
		}

		Write(responseBuffer, answerResourceRecord.Type)
		Write(responseBuffer, answerResourceRecord.Class)
		Write(responseBuffer, answerResourceRecord.TimeToLive)
		Write(responseBuffer, answerResourceRecord.ResourceDataLength)
		Write(responseBuffer, answerResourceRecord.ResourceData)
	}

	for _, authorityResourceRecord := range authorityResourceRecords {
		err = writeDomainName(responseBuffer, authorityResourceRecord.DomainName)

		if err != nil {
			log.Println("Error writing to buffer: ", err.Error())
		}

		Write(responseBuffer, authorityResourceRecord.Type)
		Write(responseBuffer, authorityResourceRecord.Class)
		Write(responseBuffer, authorityResourceRecord.TimeToLive)
		Write(responseBuffer, authorityResourceRecord.ResourceDataLength)
		Write(responseBuffer, authorityResourceRecord.ResourceData)
	}

	for _, additionalResourceRecord := range additionalResourceRecords {
		err = writeDomainName(responseBuffer, additionalResourceRecord.DomainName)

		if err != nil {
			log.Println("Error writing to buffer: ", err.Error())
		}

		Write(responseBuffer, additionalResourceRecord.Type)
		Write(responseBuffer, additionalResourceRecord.Class)
		Write(responseBuffer, additionalResourceRecord.TimeToLive)
		Write(responseBuffer, additionalResourceRecord.ResourceDataLength)
		Write(responseBuffer, additionalResourceRecord.ResourceData)
	}

	serverConn.WriteToUDP(responseBuffer.Bytes(), clientAddr)
}

func main() {

	port := flag.String("port", "1053", "port to listen on")
	flag.UintVar(&ExpiryTimeInSeconds, "expiry", 1800, "expiry time in seconds")
	
	flag.Parse()

	serverAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+ *port)


	if err != nil {
		log.Println("Error resolving UDP address: ", err.Error())
		os.Exit(1)
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)

	if err != nil {
		log.Println("Error listening: ", err.Error())
		os.Exit(1)
	}

	redisClient = redis.NewClient(&redis.Options{
	    Addr: "localhost:6379",
	    Password: "",
	    DB: 0,
	})

	log.Println("Listening at: ", serverAddr)

	defer serverConn.Close()

	for {
		requestBytes := make([]byte, UDPMaxMessageSizeBytes)

		_, clientAddr, err := serverConn.ReadFromUDP(requestBytes)

		if err != nil {
			log.Println("Error receiving: ", err.Error())
		} else {
			log.Println("Received request from ", clientAddr)
			go handleDNSClient(requestBytes, serverConn, clientAddr) // array is value type (call-by-value), i.e. copied
		}
	}
}
