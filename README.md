# humandns53

A reference implementation of humandns dns server: the dns server part of [humandns](https://github.com/h4sh5/humandns)



Run
---

```
$ go run . &
Listening at:  :1053

$ dig ack-ack-ack-low.ip4  @localhost -p 1053

; <<>> DiG 9.16.31-RH <<>> ack-ack-ack-low.ip4 @localhost -p 1053
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 23558
;; flags: qr; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 0

;; QUESTION SECTION:
;ack-ack-ack-low.ip4.		IN	A

;; ANSWER SECTION:
ack-ack-ack-low.ip4.	31337	IN	A	127.0.0.1

;; Query time: 0 msec
;; SERVER: ::1#1053(::1)
;; WHEN: Sun Feb 12 00:03:47 AEST 2023
;; MSG SIZE  rcvd: 72

```

Concepts
--------

* Go structs and methods ([Structs Instead of Classes - OOP in Go])
* Goroutines ([Rob Pike - 'Concurrency Is Not Parallelism'])
* Go slices (Go's dynamic lists)
* Efficiently writing to and reading from structs using binary.Read() and binary.Write() respectively
* DNS protocol ([RFC 1035: Domain Names - Implementation and Specification])

TODO
----

* Add support for IPv6 records


Links
-----

* [RFC 1035: Domain Names - Implementation and Specification]
* [DNS Query Message Format]
* [Wireshark]
* [Structs Instead of Classes - OOP in Go]
* [Rob Pike - 'Concurrency Is Not Parallelism']

[Authoritative vs. Recursive DNS Servers: What's The Difference]: http://social.dnsmadeeasy.com/blog/authoritative-vs-recursive-dns-servers-whats-the-difference/
[CoreDNS]: https://coredns.io/
[Go DNS]: https://github.com/miekg/dns
[package dnsmessage]: https://godoc.org/golang.org/x/net/dns/dnsmessage
[r/golang]: https://www.reddit.com/r/golang/comments/c3n7hl/simple_dns_server_implemented_in_go/
[go-nuts]: https://groups.google.com/d/msgid/golang-nuts/9d6801ae-5725-4152-83cf-33e63219da70%40googlegroups.com
[DNS Message Compression]: http://www.tcpipguide.com/free/t_DNSNameNotationandMessageCompressionTechnique-2.htm
[knome]: https://www.reddit.com/r/golang/comments/c3n7hl/simple_dns_server_implemented_in_go/erseh68?utm_source=share&utm_medium=web2x
[RFC 1035: Domain Names - Implementation and Specification]: https://www.ietf.org/rfc/rfc1035.txt
[DNS Query Message Format]: http://www.firewall.cx/networking-topics/protocols/domain-name-system-dns/160-protocols-dns-query.html
[Wireshark]: https://www.wireshark.org/
[Structs Instead of Classes - OOP in Go]: https://golangbot.com/structs-instead-of-classes/
[Rob Pike - 'Concurrency Is Not Parallelism']: https://www.youtube.com/watch?v=cN_DpYBzKso
