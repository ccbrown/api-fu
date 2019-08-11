module github.com/ccbrown/api-fu/examples/chat

go 1.12

require (
	github.com/ccbrown/api-fu v0.0.0
	github.com/ccbrown/keyvaluestore v0.0.0-20190807034003-d24ee7ef9cb1
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.3
	github.com/sirupsen/logrus v1.4.2
	github.com/vmihailenco/msgpack v4.0.4+incompatible
	golang.org/x/crypto v0.0.0-20190605123033-f99c8df09eb5
	google.golang.org/appengine v1.6.1 // indirect
)

replace github.com/ccbrown/api-fu => ../../
