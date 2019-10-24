package ws

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"github.com/Allenxuxu/toolkit/convert"
)

const (
	nonceSize = 24 // base64.StdEncoding.EncodedLen(nonceKeySize)

	// RFC6455: The value of this header field is constructed by concatenating
	// /key/, defined above in step 4 in Section 4.2.2, with the string
	// "258EAFA5- E914-47DA-95CA-C5AB0DC85B11", taking the SHA-1 hash of this
	// concatenated value to obtain a 20-byte value and base64- encoding (see
	// Section 4 of [RFC4648]) this 20-byte hash.
	acceptSize = 28 // base64.StdEncoding.EncodedLen(sha1.Size)
)

// initAcceptFromNonce fills given slice with accept bytes generated from given
// nonce bytes. Given buffer should be exactly acceptSize bytes.
func initAcceptFromNonce(accept, nonce []byte) {
	const magic = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	if len(accept) != acceptSize {
		panic("accept buffer is invalid")
	}
	if len(nonce) != nonceSize {
		panic("nonce is invalid")
	}

	p := make([]byte, nonceSize+len(magic))
	copy(p[:nonceSize], nonce)
	copy(p[nonceSize:], magic)

	sum := sha1.Sum(p)
	base64.StdEncoding.Encode(accept, sum[:])
}

func writeAccept(bw *bufio.Writer, nonce []byte) (int, error) {
	accept := make([]byte, acceptSize)
	initAcceptFromNonce(accept, nonce)
	// NOTE: write accept bytes as a string to prevent heap allocation –
	// WriteString() copy given string into its inner buffer, unlike Write()
	// which may write p directly to the underlying io.Writer – which in turn
	// will lead to p escape.
	return bw.WriteString(convert.BytesToString(accept))
}
