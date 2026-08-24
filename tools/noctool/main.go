package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	Version   = "unknown"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var (
	endpointSeal         = "/api/v1/pdf/sign"
	endpointValidate     = "/api/v1/pdf/validate"
	endpointGetSealedPDF = "/api/v1/pdf/%s"

	// Small PDF (~350 bytes) - simple single page
	pdfSmall = "JVBERi0xLjAKMSAwIG9iajw8L1R5cGUvQ2F0YWxvZy9QYWdlcyAyIDAgUj4-ZW5kb2JqCjIgMCBvYmo8PC9UeXBlL1BhZ2VzL0tpZHNbMyAwIFJdL0NvdW50IDE-PmVuZG9iagozIDAgb2JqPDwvVHlwZS9QYWdlL1BhcmVudCAyIDAgUi9SZXNvdXJjZXM8PC9Gb250PDwvRjE8PC9UeXBlL0ZvbnQvU3VidHlwZS9UeXBlMS9CYXNlRm9udC9IZWx2ZXRpY2E-Pj4-Pj4vTWVkaWFCb3hbMCAwIDIwMCA1MF0vQ29udGVudHMgNCAwIFI-PmVuZG9iago0IDAgb2JqPDwvTGVuZ3RoIDUyPj5zdHJlYW0KQlQgL0YxIDEwIFRmIDUgMjAgVGQgKFNVTkVUIEVkdVNlYWwgdGVzdCBQREYpIFRqIEVUCmVuZHN0cmVhbQplbmRvYmoKeHJlZgowIDUKMDAwMDAwMDAwMCA2NTUzNSBmIAowMDAwMDAwMDA5IDAwMDAwIG4gCjAwMDAwMDAwNTIgMDAwMDAgbiAKMDAwMDAwMDEwMSAwMDAwMCBuIAowMDAwMDAwMjUxIDAwMDAwIG4gCnRyYWlsZXI8PC9Sb290IDEgMCBSL1NpemUgNT4-CnN0YXJ0eHJlZgozNDgKJSVFT0YK"

	// Medium PDF (~2KB) - multi-page with more content
	pdfMedium = "JVBERi0xLjQKMSAwIG9iajw8L1R5cGUvQ2F0YWxvZy9QYWdlcyAyIDAgUj4-ZW5kb2JqCjIgMCBvYmo8PC9UeXBlL1BhZ2VzL0tpZHNbMyAwIFIgNSAwIFIgNyAwIFJdL0NvdW50IDM-PmVuZG9iagozIDAgb2JqPDwvVHlwZS9QYWdlL1BhcmVudCAyIDAgUi9SZXNvdXJjZXM8PC9Gb250PDwvRjE8PC9UeXBlL0ZvbnQvU3VidHlwZS9UeXBlMS9CYXNlRm9udC9IZWx2ZXRpY2E-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDQgMCBSPj5lbmRvYmoKNCAwIG9iajw8L0xlbmd0aCAyMDA-PnN0cmVhbQpCVCAvRjEgMTIgVGYgNTAgNzAwIFRkIChTVU5FVCBFZHVTZWFsIE1lZGl1bSBUZXN0IFBERiAtIFBhZ2UgMSkgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDY1MCBUZCAoVGhpcyBpcyBhIG1lZGl1bSBzaXplZCB0ZXN0IGRvY3VtZW50IHdpdGggbXVsdGlwbGUgcGFnZXMuKSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjAwIFRkIChJdCBjb250YWlucyBtb3JlIGNvbnRlbnQgdGhhbiB0aGUgc21hbGwgUERGLikgVGogRVQKZW5kc3RyZWFtCmVuZG9iago1IDAgb2JqPDwvVHlwZS9QYWdlL1BhcmVudCAyIDAgUi9SZXNvdXJjZXM8PC9Gb250PDwvRjE8PC9UeXBlL0ZvbnQvU3VidHlwZS9UeXBlMS9CYXNlRm9udC9IZWx2ZXRpY2E-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDYgMCBSPj5lbmRvYmoKNiAwIG9iajw8L0xlbmd0aCA4MD4-c3RyZWFtCkJUIC9GMSAxMiBUZiA1MCA3MDAgVGQgKFNVTkVUIEVkdVNlYWwgTWVkaXVtIFRlc3QgUERGIC0gUGFnZSAyKSBUaiBFVAplbmRzdHJlYW0KZW5kb2JqCjcgMCBvYmo8PC9UeXBlL1BhZ2UvUGFyZW50IDIgMCBSL1Jlc291cmNlczw8L0ZvbnQ8PC9GMTw8L1R5cGUvRm9udC9TdWJ0eXBlL1R5cGUxL0Jhc2VGb250L0hlbHZldGljYT4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDggMCBSPj5lbmRvYmoKOCAwIG9iajw8L0xlbmd0aCA4MD4-c3RyZWFtCkJUIC9GMSAxMiBUZiA1MCA3MDAgVGQgKFNVTkVUIEVkdVNlYWwgTWVkaXVtIFRlc3QgUERGIC0gUGFnZSAzKSBUaiBFVAplbmRzdHJlYW0KZW5kb2JqCnhyZWYKMCA5CjAwMDAwMDAwMDAgNjU1MzUgZiAKMDAwMDAwMDAwOSAwMDAwMCBuIAowMDAwMDAwMDUyIDAwMDAwIG4gCjAwMDAwMDAxMTMgMDAwMDAgbiAKMDAwMDAwMDI2MyAwMDAwMCBuIAowMDAwMDAwNTE0IDAwMDAwIG4gCjAwMDAwMDA2NjQgMDAwMDAgbiAKMDAwMDAwMDc5MyAwMDAwMCBuIAowMDAwMDAwOTQzIDAwMDAwIG4gCnRyYWlsZXI8PC9Sb290IDEgMCBSL1NpemUgOT4-CnN0YXJ0eHJlZgoxMDcyCiUlRU9GCg=="

	// Big PDF (~10KB) - many pages with substantial content
	pdfBig = "JVBERi0xLjQKMSAwIG9iajw8L1R5cGUvQ2F0YWxvZy9QYWdlcyAyIDAgUj4-ZW5kb2JqCjIgMCBvYmo8PC9UeXBlL1BhZ2VzL0tpZHNbMyAwIFIgNSAwIFIgNyAwIFIgOSAwIFIgMTEgMCBSIDEzIDAgUiAxNSAwIFIgMTcgMCBSIDE5IDAgUiAyMSAwIFJdL0NvdW50IDEwPj5lbmRvYmoKMyAwIG9iajw8L1R5cGUvUGFnZS9QYXJlbnQgMiAwIFIvUmVzb3VyY2VzPDwvRm9udDw8L0YxPDwvVHlwZS9Gb250L1N1YnR5cGUvVHlwZTEvQmFzZUZvbnQvSGVsdmV0aWNhPj4vRjI8PC9UeXBlL0ZvbnQvU3VidHlwZS9UeXBlMS9CYXNlRm9udC9IZWx2ZXRpY2EtQm9sZD4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDQgMCBSPj5lbmRvYmoKNCAwIG9iajw8L0xlbmd0aCA1MDA-PnN0cmVhbQpCVCAvRjIgMTggVGYgNTAgNzIwIFRkIChTVU5FVCBFZHVTZWFsIEJJRyBUZXN0IFBERiAtIFBhZ2UgMSkgVGogRVQKQlQgL0YxIDEyIFRmIDUwIDY1MCBUZCAoVGhpcyBpcyBhIGxhcmdlIHRlc3QgZG9jdW1lbnQgd2l0aCBtdWx0aXBsZSBwYWdlcyBhbmQgc3Vic3RhbnRpYWwgY29udGVudC4pIFRqIEVUCkJUIC9GMSAxMCBUZiA1MCA2MDAgVGQgKFRoaXMgZG9jdW1lbnQgaXMgZGVzaWduZWQgdG8gdGVzdCB0aGUgc3lzdGVtIHdpdGggbGFyZ2VyIGZpbGVzLikgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDU1MCBUZCAoSXQgY29udGFpbnMgbXVsdGlwbGUgcGFnZXMgb2YgY29udGVudCB0byBzaW11bGF0ZSByZWFsLXdvcmxkIHVzYWdlLikgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDUwMCBUZCAoRWFjaCBwYWdlIGhhcyBhZGRpdGlvbmFsIHRleHQgdG8gaW5jcmVhc2UgdGhlIGZpbGUgc2l6ZS4pIFRqIEVUCmVuZHN0cmVhbQplbmRvYmoKNSAwIG9iajw8L1R5cGUvUGFnZS9QYXJlbnQgMiAwIFIvUmVzb3VyY2VzPDwvRm9udDw8L0YxPDwvVHlwZS9Gb250L1N1YnR5cGUvVHlwZTEvQmFzZUZvbnQvSGVsdmV0aWNhPj4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDYgMCBSPj5lbmRvYmoKNiAwIG9iajw8L0xlbmd0aCAzMDA-PnN0cmVhbQpCVCAvRjEgMTQgVGYgNTAgNzIwIFRkIChTVU5FVCBFZHVTZWFsIEJJRyBUZXN0IFBERiAtIFBhZ2UgMikgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDY1MCBUZCAoQ29udGludWluZyB3aXRoIG1vcmUgY29udGVudCBvbiBwYWdlIDIuKSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjAwIFRkIChUaGlzIHBhZ2UgYWRkcyBhZGRpdGlvbmFsIHRleHQgZm9yIHRlc3RpbmcgcHVycG9zZXMuKSBUaiBFVAplbmRzdHJlYW0KZW5kb2JqCjcgMCBvYmo8PC9UeXBlL1BhZ2UvUGFyZW50IDIgMCBSL1Jlc291cmNlczw8L0ZvbnQ8PC9GMTw8L1R5cGUvRm9udC9TdWJ0eXBlL1R5cGUxL0Jhc2VGb250L0hlbHZldGljYT4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDggMCBSPj5lbmRvYmoKOCAwIG9iajw8L0xlbmd0aCAzMDA-PnN0cmVhbQpCVCAvRjEgMTQgVGYgNTAgNzIwIFRkIChTVU5FVCBFZHVTZWFsIEJJRyBUZXN0IFBERiAtIFBhZ2UgMykgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDY1MCBUZCAoUGFnZSAzIGNvbnRhaW5zIG1vcmUgZGF0YSB0byBpbmNyZWFzZSB0aGUgUERGIHNpemUuKSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjAwIFRkIChXZSBhcmUgdGVzdGluZyB3aXRoIGxhcmdlciBmaWxlcyBmb3IgcmVhbGlzbS4pIFRqIEVUCmVuZHN0cmVhbQplbmRvYmoKOSAwIG9iajw8L1R5cGUvUGFnZS9QYXJlbnQgMiAwIFIvUmVzb3VyY2VzPDwvRm9udDw8L0YxPDwvVHlwZS9Gb250L1N1YnR5cGUvVHlwZTEvQmFzZUZvbnQvSGVsdmV0aWNhPj4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDEwIDAgUj4-ZW5kb2JqCjEwIDAgb2JqPDwvTGVuZ3RoIDMwMD4-c3RyZWFtCkJUIC9GMSAxNCBUZiA1MCA3MjAgVGQgKFNVTkVUIEVkdVNlYWwgQklHIFRlc3QgUERGIC0gUGFnZSA0KSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjUwIFRkIChQYWdlIDQgd2l0aCBhZGRpdGlvbmFsIGNvbnRlbnQgZm9yIHRlc3RpbmcuKSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjAwIFRkIChNb3JlIHRleHQgdG8gaW5jcmVhc2UgdGhlIG92ZXJhbGwgZmlsZSBzaXplLikgVGogRVQKZW5kc3RyZWFtCmVuZG9iagoxMSAwIG9iajw8L1R5cGUvUGFnZS9QYXJlbnQgMiAwIFIvUmVzb3VyY2VzPDwvRm9udDw8L0YxPDwvVHlwZS9Gb250L1N1YnR5cGUvVHlwZTEvQmFzZUZvbnQvSGVsdmV0aWNhPj4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDEyIDAgUj4-ZW5kb2JqCjEyIDAgb2JqPDwvTGVuZ3RoIDMwMD4-c3RyZWFtCkJUIC9GMSAxNCBUZiA1MCA3MjAgVGQgKFNVTkVUIEVkdVNlYWwgQklHIFRlc3QgUERGIC0gUGFnZSA1KSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjUwIFRkIChQYWdlIDUgY29udGludWVzIHRoZSBwYXR0ZXJuIG9mIGFkZGluZyBjb250ZW50LikgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDYwMCBUZCAoVGhpcyBoZWxwcyBjcmVhdGUgYSBzdWJzdGFudGlhbCB0ZXN0IGRvY3VtZW50LikgVGogRVQKZW5kc3RyZWFtCmVuZG9iagoxMyAwIG9iajw8L1R5cGUvUGFnZS9QYXJlbnQgMiAwIFIvUmVzb3VyY2VzPDwvRm9udDw8L0YxPDwvVHlwZS9Gb250L1N1YnR5cGUvVHlwZTEvQmFzZUZvbnQvSGVsdmV0aWNhPj4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDE0IDAgUj4-ZW5kb2JqCjE0IDAgb2JqPDwvTGVuZ3RoIDMwMD4-c3RyZWFtCkJUIC9GMSAxNCBUZiA1MCA3MjAgVGQgKFNVTkVUIEVkdVNlYWwgQklHIFRlc3QgUERGIC0gUGFnZSA2KSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjUwIFRkIChQYWdlIDYgYWRkcyBtb3JlIGRhdGEgdG8gdGhlIGRvY3VtZW50LikgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDYwMCBUZCAoRWFjaCBwYWdlIGNvbnRyaWJ1dGVzIHRvIHRoZSB0b3RhbCBzaXplLikgVGogRVQKZW5kc3RyZWFtCmVuZG9iagoxNSAwIG9iajw8L1R5cGUvUGFnZS9QYXJlbnQgMiAwIFIvUmVzb3VyY2VzPDwvRm9udDw8L0YxPDwvVHlwZS9Gb250L1N1YnR5cGUvVHlwZTEvQmFzZUZvbnQvSGVsdmV0aWNhPj4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDE2IDAgUj4-ZW5kb2JqCjE2IDAgb2JqPDwvTGVuZ3RoIDMwMD4-c3RyZWFtCkJUIC9GMSAxNCBUZiA1MCA3MjAgVGQgKFNVTkVUIEVkdVNlYWwgQklHIFRlc3QgUERGIC0gUGFnZSA3KSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjUwIFRkIChQYWdlIDcgd2l0aCBhZGRpdGlvbmFsIHRleHQgZm9yIHRlc3RpbmcgcHVycG9zZXMuKSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjAwIFRkIChXZSdyZSBidWlsZGluZyBhIGxhcmdlciBQREYgZm9yIHN0cmVzcyB0ZXN0aW5nLikgVGogRVQKZW5kc3RyZWFtCmVuZG9iagoxNyAwIG9iajw8L1R5cGUvUGFnZS9QYXJlbnQgMiAwIFIvUmVzb3VyY2VzPDwvRm9udDw8L0YxPDwvVHlwZS9Gb250L1N1YnR5cGUvVHlwZTEvQmFzZUZvbnQvSGVsdmV0aWNhPj4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDE4IDAgUj4-ZW5kb2JqCjE4IDAgb2JqPDwvTGVuZ3RoIDMwMD4-c3RyZWFtCkJUIC9GMSAxNCBUZiA1MCA3MjAgVGQgKFNVTkVUIEVkdVNlYWwgQklHIFRlc3QgUERGIC0gUGFnZSA4KSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjUwIFRkIChQYWdlIDggY29udGludWVzIHRoZSBwYXR0ZXJuIHdpdGggbW9yZSBjb250ZW50LikgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDYwMCBUZCAoVGhpcyBkb2N1bWVudCBpcyBnZXR0aW5nIHF1aXRlIHN1YnN0YW50aWFsIG5vdy4pIFRqIEVUCmVuZHN0cmVhbQplbmRvYmoKMTkgMCBvYmo8PC9UeXBlL1BhZ2UvUGFyZW50IDIgMCBSL1Jlc291cmNlczw8L0ZvbnQ8PC9GMTw8L1R5cGUvRm9udC9TdWJ0eXBlL1R5cGUxL0Jhc2VGb250L0hlbHZldGljYT4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDIwIDAgUj4-ZW5kb2JqCjIwIDAgb2JqPDwvTGVuZ3RoIDMwMD4-c3RyZWFtCkJUIC9GMSAxNCBUZiA1MCA3MjAgVGQgKFNVTkVUIEVkdVNlYWwgQklHIFRlc3QgUERGIC0gUGFnZSA5KSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjUwIFRkIChQYWdlIDkgYWRkcyBldmVuIG1vcmUgY29udGVudCB0byB0aGUgZmlsZS4pIFRqIEVUCkJUIC9GMSAxMCBUZiA1MCA2MDAgVGQgKFdlJ3JlIGFsbW9zdCBhdCB0aGUgZW5kIG9mIHRoaXMgdGVzdCBkb2N1bWVudC4pIFRqIEVUCmVuZHN0cmVhbQplbmRvYmoKMjEgMCBvYmo8PC9UeXBlL1BhZ2UvUGFyZW50IDIgMCBSL1Jlc291cmNlczw8L0ZvbnQ8PC9GMTw8L1R5cGUvRm9udC9TdWJ0eXBlL1R5cGUxL0Jhc2VGb250L0hlbHZldGljYT4-Pj4-Pj4vTWVkaWFCb3hbMCAwIDYxMiA3OTJdL0NvbnRlbnRzIDIyIDAgUj4-ZW5kb2JqCjIyIDAgb2JqPDwvTGVuZ3RoIDMwMD4-c3RyZWFtCkJUIC9GMSAxNCBUZiA1MCA3MjAgVGQgKFNVTkVUIEVkdVNlYWwgQklHIFRlc3QgUERGIC0gUGFnZSAxMCkgVGogRVQKQlQgL0YxIDEwIFRmIDUwIDY1MCBUZCAoRmluYWwgcGFnZSBvZiB0aGUgYmlnIHRlc3QgZG9jdW1lbnQuKSBUaiBFVApCVCAvRjEgMTAgVGYgNTAgNjAwIFRkIChUaGlzIGNvbXBsZXRlcyBvdXIgbGFyZ2UgUERGIGZvciB0ZXN0aW5nLikgVGogRVQKZW5kc3RyZWFtCmVuZG9iagp4cmVmCjAgMjMKMDAwMDAwMDAwMCA2NTUzNSBmIAowMDAwMDAwMDA5IDAwMDAwIG4gCjAwMDAwMDAwNTIgMDAwMDAgbiAKMDAwMDAwMDE0NyAwMDAwMCBuIAowMDAwMDAwMzc1IDAwMDAwIG4gCjAwMDAwMDA5MjYgMDAwMDAgbiAKMDAwMDAwMTA3NiAwMDAwMCBuIAowMDAwMDAxNDI1IDAwMDAwIG4gCjAwMDAwMDE1NzUgMDAwMDAgbiAKMDAwMDAwMTkyNCAwMDAwMCBuIAowMDAwMDAyMDc1IDAwMDAwIG4gCjAwMDAwMDI0MjYgMDAwMDAgbiAKMDAwMDAwMjU3NyAwMDAwMCBuIAowMDAwMDAyOTI4IDAwMDAwIG4gCjAwMDAwMDMwNzkgMDAwMDAgbiAKMDAwMDAwMzQzMCAwMDAwMCBuIAowMDAwMDAzNTgxIDAwMDAwIG4gCjAwMDAwMDM5MzIgMDAwMDAgbiAKMDAwMDAwNDA4MyAwMDAwMCBuIAowMDAwMDA0NDM0IDAwMDAwIG4gCjAwMDAwMDQ1ODUgMDAwMDAgbiAKMDAwMDAwNDkzNyAwMDAwMCBuIAowMDAwMDA1MDg4IDAwMDAwIG4gCnRyYWlsZXI8PC9Sb290IDEgMCBSL1NpemUgMjM-PgpzdGFydHhyZWYKNTQ0MAolJUVPRgo="
)

type validationResponse struct {
	Data struct {
		ValidationBackend string `json:"validation_backend"`
		IntactSignature   bool   `json:"intact_signature"`
		ValidSignature    bool   `json:"valid_signature"`
		TransactionID     string `json:"transaction_id"`
	} `json:"data"`
}

// authRequestBody is the new mTLS-based auth request
type authRequestBody struct {
	AccessToken []authAccessToken `json:"access_token"`
	Client      authClient        `json:"client"`
}

type authAccessToken struct {
	Flags []string `json:"flags"`
}

type authClient struct {
	Key authClientKey `json:"key"`
}

type authClientKey struct {
	Proof string `json:"proof"`
	Cert  string `json:"cert"`
}

type StormMode struct {
	Enabled          bool `json:"enabled" yaml:"enabled"`
	MaxRetries       int  `json:"max_retries" yaml:"max_retries"`
	RetryWaitMs      int  `json:"retry_wait_ms" yaml:"retry_wait_ms"`
	UploadIntervalMs int  `json:"upload_interval_ms" yaml:"upload_interval_ms"`
	MaxUploads       int  `json:"max_uploads" yaml:"max_uploads"`
	FetchTimeoutSec  int  `json:"fetch_timeout_sec" yaml:"fetch_timeout_sec"`
}

type Config struct {
	OAuth         map[string]any `json:"oauth,omitempty" yaml:"oauth,omitempty"`
	Env           string         `json:"env,omitempty" yaml:"env,omitempty"`
	TestCase      string         `json:"testcase,omitempty" yaml:"testcase,omitempty"`
	Save          bool           `json:"save,omitempty" yaml:"save,omitempty"`
	ClientCert    string         `json:"client_cert,omitempty" yaml:"client_cert,omitempty"`
	ClientCertKey string         `json:"client_cert_key,omitempty" yaml:"client_cert_key,omitempty"`
	PDFSize       string         `json:"pdf_size,omitempty" yaml:"pdf_size,omitempty"`
	Storm         StormMode      `json:"storm,omitempty" yaml:"storm,omitempty"`
}

type fetchResponse struct {
	Data   fetchResponseData `json:"data"`
	Status string            `json:"status,omitempty"`
}

type fetchResponseData struct {
	TransactionID string `json:"transaction_id"`
	Data          string `json:"data"` // base64-encoded PDF
	SealerBackend string `json:"sealer_backend"`
	SealedPDF     string `json:"sealed_pdf,omitempty"` // alternative field name
	Status        string `json:"status,omitempty"`
}

type stormStats struct {
	TotalUploads      int
	SuccessfulUploads int
	FailedUploads     int
	TotalRetries      int
	StartTime         time.Time
}

type Client struct {
	httpClient           *http.Client
	env                  string
	serviceBaseURL       string
	accessTransactionURL string
	accessToken          string
	tokenExpiresAt       time.Time
	testCase             string
	sealedPDF            string
	transactionID        string
	validationResponse   validationResponse
	config               Config
	clientCertPEM        string
	stats                stormStats
	errorLogFile         *os.File
	validatePDFPath      string
}

func main() {
	versionFlag := flag.Bool("version", false, "print version and exit")
	configFlag := flag.String("config", "", "path to YAML config file")
	validatePDFFlag := flag.String("validate-pdf", "", "path to an existing PDF to submit to /api/v1/pdf/validate (skips testcase and storm mode)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("%s git:%s built:%s\n", Version, GitCommit, BuildDate)
		os.Exit(0)
	}

	config, err := loadAccessRequestBody(*configFlag)
	if err != nil {
		fmt.Printf("\033[31m✗\033[0m could not load access request body: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("flags", "env:", config.Env, "testcase:", config.TestCase)

	if config.ClientCert == "" || config.ClientCertKey == "" {
		fmt.Println("\033[31m✗\033[0m Error: client_cert and client_cert_key are required in config file")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("\033[32m✓\033[0m Loading client certificate from %s and %s\n", config.ClientCert, config.ClientCertKey)
	clientCert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientCertKey)
	if err != nil {
		fmt.Printf("\033[31m✗\033[0m could not load client key pair: %v\n", err)
		os.Exit(1)
	}

	// Read the client certificate PEM for the auth request body
	certPEMBytes, err := os.ReadFile(filepath.Clean(config.ClientCert))
	if err != nil {
		fmt.Printf("\033[31m✗\033[0m could not read client certificate file: %v\n", err)
		os.Exit(1)
	}
	clientCertPEM := string(certPEMBytes)

	// Parse and display client certificate details
	if len(clientCert.Certificate) > 0 {
		_, err := x509.ParseCertificate(clientCert.Certificate[0])
		if err != nil {
			fmt.Printf("Warning: Could not parse client certificate: %v\n", err)
		}
	}

	// Load ISRG Root X1 (Let's Encrypt root cert) for server verification
	pool := x509.NewCertPool()
	isrgCert, err := os.ReadFile("/etc/ssl/certs/ISRG_Root_X1.pem")
	if err != nil {
		fmt.Printf("could not read ISRG Root X1 certificate: %v\n", err)
		os.Exit(1)
	}
	if ok := pool.AppendCertsFromPEM(isrgCert); ok {
		fmt.Println("\033[32m✓\033[0m ISRG Root X1 certificate loaded successfully")
	} else {
		fmt.Println("\033[31m✗\033[0m Warning: Failed to add ISRG Root X1 certificate")
		os.Exit(1)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			Certificates:       []tls.Certificate{clientCert},
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
			InsecureSkipVerify: false,
			Renegotiation:      tls.RenegotiateOnceAsClient,
			GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return &clientCert, nil
			},
		},
		Proxy: http.ProxyFromEnvironment,
	}

	client := Client{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		env:                config.Env,
		testCase:           config.TestCase,
		validationResponse: validationResponse{},
		config:             config,
		clientCertPEM:      clientCertPEM,
		validatePDFPath:    *validatePDFFlag,
	}

	switch client.env {
	case "test":
		client.serviceBaseURL = "https://test-api.eduseal.sunet.se"
		client.accessTransactionURL = "https://auth-test.sunet.se/transaction"
	case "qa":
		client.serviceBaseURL = "https://qa-api.eduseal.sunet.se"
		client.accessTransactionURL = "https://auth-test.sunet.se/transaction"
	case "prod":
		client.serviceBaseURL = "https://api.eduseal.sunet.se"
		client.accessTransactionURL = "https://auth.sunet.se/transaction"
	default:
		fmt.Printf("\033[31m✗\033[0m unknown environment: %s\n", client.env)
		os.Exit(1)
	}

	if err := client.getAccessToken(); err != nil {
		fmt.Printf("\033[31m✗\033[0m could not get access token: %v\n", err)
		os.Exit(1)
	}

	if client.validatePDFPath != "" {
		if err := client.validateExistingPDF(); err != nil {
			fmt.Printf("\033[31m✗\033[0m could not validate PDF: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check if storm mode is enabled
	if config.Storm.Enabled {
		fmt.Println("\033[33m⚡ Storm mode enabled\033[0m")
		if err := client.runStormMode(); err != nil {
			fmt.Printf("\033[31m✗\033[0m storm mode error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	switch client.testCase {
	case "ladok":
		if err := client.checkPDFSealing(config.Save); err != nil {
			fmt.Printf("\033[31m✗\033[0m could not seal PDF: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("\033[31m✗\033[0m unknown test case: %s\n", client.testCase)
		os.Exit(1)
	}
}

func (c *Client) getPDF() string {
	switch c.config.PDFSize {
	case "medium":
		return pdfMedium
	case "big":
		return pdfBig
	case "small":
		fallthrough
	default:
		return pdfSmall
	}
}

func loadAccessRequestBody(configPath string) (Config, error) {
	if configPath == "" {
		return Config{}, fmt.Errorf("-config flag is required")
	}

	configData, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config YAML: %v", err)
	}

	return config, nil
}

func (c *Client) getAccessToken() error {
	var requestBody []byte
	var err error

	if c.config.OAuth != nil {
		requestBody, err = json.Marshal(c.config.OAuth)
	} else {
		body := authRequestBody{
			AccessToken: []authAccessToken{
				{Flags: []string{"bearer"}},
			},
			Client: authClient{
				Key: authClientKey{
					Proof: "mtls",
					Cert:  c.clientCertPEM,
				},
			},
		}
		requestBody, err = json.Marshal(body)
	}
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.accessTransactionURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("\n\033[31m✗\033[0m Error making request: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("\033[31m✗\033[0m Error Response Body: %s\n", string(bodyBytes))
		fmt.Printf("\033[31m✗\033[0m Error Response Body Size: %d bytes\n", len(bodyBytes))
		fmt.Printf("=== END ACCESS TOKEN REQUEST (FAILED) ===\n\n")
		return fmt.Errorf("failed to get token, status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return err
	}

	accessToken := payload["access_token"].(map[string]any)

	c.accessToken = accessToken["value"].(string)

	// Extract expires_in to determine when to refresh the token
	if expiresIn, ok := accessToken["expires_in"].(float64); ok {
		c.tokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
		fmt.Printf("\033[32m✓\033[0m Access token valid for %.0f seconds (expires at %s)\n",
			expiresIn, c.tokenExpiresAt.Format("15:04:05"))
	} else {
		// Default to 1 hour if expires_in not provided
		c.tokenExpiresAt = time.Now().Add(1 * time.Hour)
		fmt.Printf("\033[33m⚠\033[0m No expires_in found, defaulting to 1 hour\n")
	}

	return nil
}

func (c *Client) checkPDFSealing(shouldSave bool) error {
	fmt.Println("Sealing PDF...")

	requestBody := map[string]any{
		"pdf": c.getPDF(),
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.serviceBaseURL+endpointSeal, bytes.NewBuffer(requestBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to seal PDF, status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	responsePayload := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &responsePayload); err != nil {
		return err
	}

	data := responsePayload["data"].(map[string]any)

	var ok bool
	c.transactionID, ok = data["transaction_id"].(string)
	if !ok {
		return fmt.Errorf("transaction_id not found in response")
	}

	fmt.Printf("  Transaction ID: %s\n", c.transactionID)

	if err := c.fetchSealedPDF(); err != nil {
		return err
	}

	if err := c.validatePDF(); err != nil {
		return err
	}

	fmt.Println("\033[32m\u2713\033[0m PDF validated successfully")
	fmt.Printf("  Transaction ID: %s\n", c.validationResponse.Data.TransactionID)
	fmt.Printf("  Backend: %s\n", c.validationResponse.Data.ValidationBackend)
	fmt.Printf("  Intact Signature: %v\n", c.validationResponse.Data.IntactSignature)
	fmt.Printf("  Valid Signature: %v\n", c.validationResponse.Data.ValidSignature)

	if err := c.savePDF(shouldSave); err != nil {
		return err
	}

	return nil
}

func (c *Client) savePDF(shouldSave bool) error {
	if !shouldSave {
		return nil
	}

	filename := c.transactionID + ".pdf"

	pdfBytes, err := base64.StdEncoding.DecodeString(c.sealedPDF)
	if err != nil {
		return fmt.Errorf("failed to decode PDF: %v", err)
	}

	if err := os.WriteFile(filepath.Clean(filename), pdfBytes, 0600); err != nil {
		return fmt.Errorf("failed to write PDF file: %v", err)
	}

	fmt.Printf("\033[32m✓\033[0m PDF saved to %s\n", filename)
	return nil
}

func (c *Client) fetchSealedPDF() error {
	return c.fetchSealedPDFWithConfig(0)
}

func (c *Client) fetchSealedPDFWithConfig(attemptNum int) error {
	if attemptNum == 0 {
		fmt.Println("Fetching sealed PDF...")
	}

	timeout := 11 * time.Second
	if c.config.Storm.Enabled && c.config.Storm.FetchTimeoutSec > 0 {
		timeout = time.Duration(c.config.Storm.FetchTimeoutSec) * time.Second
	}

	waitTime := 500 * time.Millisecond
	if c.config.Storm.Enabled && c.config.Storm.RetryWaitMs > 0 {
		waitTime = time.Duration(c.config.Storm.RetryWaitMs) * time.Millisecond
	}

	stop := time.Now().Add(timeout)

	for time.Now().Before(stop) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(c.serviceBaseURL+endpointGetSealedPDF, c.transactionID), nil)
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if err := resp.Body.Close(); err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			time.Sleep(waitTime)
			continue
		}

		var fetchResp fetchResponse
		if err := json.Unmarshal(bodyBytes, &fetchResp); err != nil {
			return err
		}

		// Check for PDF in data.data field (primary field)
		if fetchResp.Data.Data != "" && len(fetchResp.Data.Data) > 100 {
			c.sealedPDF = fetchResp.Data.Data
			return nil
		}

		// Check for PDF in data.sealed_pdf field (alternative field)
		if fetchResp.Data.SealedPDF != "" {
			c.sealedPDF = fetchResp.Data.SealedPDF
			return nil
		}

		// If we got a 200 but no PDF, the job might still be processing
		// Check for status field to see if job is complete
		if fetchResp.Status != "" || fetchResp.Data.Status != "" {
			status := fetchResp.Status
			if status == "" {
				status = fetchResp.Data.Status
			}
			if status == "completed" || status == "done" || status == "success" {
				// Job is done but no PDF? This is an error
				return fmt.Errorf("job completed but PDF not found in response")
			}
		}

		// Not ready yet, wait and retry
		time.Sleep(waitTime)
	}

	return errors.New("timed out waiting for sealed PDF")
}

func (c *Client) validateExistingPDF() error {
	fmt.Printf("Reading PDF from %s...\n", c.validatePDFPath)

	pdfBytes, err := os.ReadFile(filepath.Clean(c.validatePDFPath))
	if err != nil {
		return fmt.Errorf("failed to read PDF file: %v", err)
	}

	c.sealedPDF = base64.StdEncoding.EncodeToString(pdfBytes)

	if err := c.validatePDF(); err != nil {
		return err
	}

	fmt.Println("\033[32m\u2713\033[0m PDF validated successfully")
	dataJSON, err := json.MarshalIndent(c.validationResponse.Data, "  ", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal validation response: %v", err)
	}
	fmt.Printf("  %s\n", dataJSON)

	return nil
}

func (c *Client) validatePDF() error {
	fmt.Println("Validating PDF...")

	requestBody := map[string]any{
		"pdf": c.sealedPDF,
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.serviceBaseURL+endpointValidate, bytes.NewBuffer(requestBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to validate PDF, status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bodyBytes, &c.validationResponse); err != nil {
		return err
	}
	return nil
}

func (c *Client) runStormMode() error {
	c.stats = stormStats{
		StartTime: time.Now(),
	}

	// Create error log file
	errorLogPath := fmt.Sprintf("storm_errors_%s.log", time.Now().Format("20060102_150405"))
	var err error
	c.errorLogFile, err = os.OpenFile(errorLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to create error log file: %v", err)
	}
	defer c.errorLogFile.Close()

	// Start token refresh goroutine for storm mode
	stopTokenRefresh := make(chan bool)
	defer func() {
		stopTokenRefresh <- true
	}()
	go c.tokenRefreshWorker(stopTokenRefresh)

	fmt.Println("\033[33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Println("\033[33m⚡ STORM MODE - Continuous Testing ⚡\033[0m")
	fmt.Println("\033[33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
	fmt.Printf("  Error Log: %s\n", errorLogPath)
	fmt.Printf("  Max Retries: %d\n", c.config.Storm.MaxRetries)
	fmt.Printf("  Retry Wait: %dms\n", c.config.Storm.RetryWaitMs)
	fmt.Printf("  Upload Interval: %dms\n", c.config.Storm.UploadIntervalMs)
	fmt.Printf("  Fetch Timeout: %ds\n", c.config.Storm.FetchTimeoutSec)
	if c.config.Storm.MaxUploads == 0 {
		fmt.Println("  Max Uploads: ∞ (unlimited)")
	} else {
		fmt.Printf("  Max Uploads: %d\n", c.config.Storm.MaxUploads)
	}
	fmt.Println("\033[33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")

	uploadInterval := time.Duration(c.config.Storm.UploadIntervalMs) * time.Millisecond
	if uploadInterval == 0 {
		uploadInterval = 1 * time.Second
	}

	for {
		c.stats.TotalUploads++
		uploadNum := c.stats.TotalUploads

		fmt.Printf("\n\033[36m[Upload #%d]\033[0m Starting at %s\n", uploadNum, time.Now().Format("15:04:05"))

		if err := c.stormUpload(uploadNum); err != nil {
			c.stats.FailedUploads++
			fmt.Printf("\033[31m✗ [Upload #%d] Failed: %v\033[0m\n", uploadNum, err)
			c.logError(uploadNum, err)
		} else {
			c.stats.SuccessfulUploads++
			fmt.Printf("\033[32m✓ [Upload #%d] Completed successfully\033[0m\n", uploadNum)
		}

		c.printStormStats()

		// Check if we should stop (0 means unlimited)
		if c.config.Storm.MaxUploads > 0 && c.stats.TotalUploads >= c.config.Storm.MaxUploads {
			fmt.Println("\n\033[33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
			fmt.Println("\033[33m⚡ Storm mode completed ⚡\033[0m")
			fmt.Println("\033[33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
			break
		}

		// Wait before next upload
		time.Sleep(uploadInterval)
	}

	return nil
}

func (c *Client) stormUpload(uploadNum int) error {
	// Seal the PDF
	fmt.Printf("  [Upload #%d] Sealing PDF...\n", uploadNum)
	requestBody := map[string]any{
		"pdf": c.getPDF(),
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal error: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.serviceBaseURL+endpointSeal, bytes.NewBuffer(requestBytes))
	if err != nil {
		return fmt.Errorf("request creation error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("seal failed with status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read error: %v", err)
	}

	responsePayload := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &responsePayload); err != nil {
		return fmt.Errorf("unmarshal error: %v", err)
	}

	data := responsePayload["data"].(map[string]any)
	var ok bool
	c.transactionID, ok = data["transaction_id"].(string)
	if !ok {
		return fmt.Errorf("transaction_id not found")
	}

	fmt.Printf("  [Upload #%d] Transaction ID: %s\n", uploadNum, c.transactionID)

	// Fetch sealed PDF with retries
	maxRetries := c.config.Storm.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	var fetchErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		c.stats.TotalRetries++
		fmt.Printf("  [Upload #%d] Fetch attempt %d/%d...\n", uploadNum, attempt, maxRetries)

		fetchErr = c.fetchSealedPDFWithConfig(attempt)
		if fetchErr == nil {
			fmt.Printf("  [Upload #%d] \033[32m✓\033[0m PDF fetched successfully\n", uploadNum)
			break
		}

		if attempt < maxRetries {
			fmt.Printf("  [Upload #%d] \033[33m⚠\033[0m Retry %d failed: %v\n", uploadNum, attempt, fetchErr)
		}
	}

	if fetchErr != nil {
		// All fetch attempts exhausted - log this specifically
		exhaustedErr := fmt.Errorf("fetch failed after %d retries: %v", maxRetries, fetchErr)
		c.logFetchAttemptsExhausted(uploadNum, maxRetries, fetchErr)
		return exhaustedErr
	}

	// Validate the PDF
	fmt.Printf("  [Upload #%d] Validating...\n", uploadNum)
	if err := c.validatePDF(); err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}

	fmt.Printf("  [Upload #%d] \033[32m✓\033[0m Validation: Intact=%v, Valid=%v\n",
		uploadNum,
		c.validationResponse.Data.IntactSignature,
		c.validationResponse.Data.ValidSignature)

	// Save if configured
	if c.config.Save {
		filename := fmt.Sprintf("storm_%d_%s.pdf", uploadNum, c.transactionID)
		pdfBytes, err := base64.StdEncoding.DecodeString(c.sealedPDF)
		if err != nil {
			return fmt.Errorf("decode failed: %v", err)
		}
		if err := os.WriteFile(filepath.Clean(filename), pdfBytes, 0600); err != nil {
			return fmt.Errorf("save failed: %v", err)
		}
		fmt.Printf("  [Upload #%d] Saved to %s\n", uploadNum, filename)
	}

	return nil
}

func (c *Client) logError(uploadNum int, err error) {
	if c.errorLogFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] Upload #%d | TransactionID: %s | Error: %v\n",
		timestamp, uploadNum, c.transactionID, err)

	if _, writeErr := c.errorLogFile.WriteString(logEntry); writeErr != nil {
		fmt.Printf("  \033[33m⚠\033[0m Failed to write to error log: %v\n", writeErr)
	}
}

func (c *Client) logFetchAttemptsExhausted(uploadNum int, maxRetries int, lastErr error) {
	if c.errorLogFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] 🔄 FETCH ATTEMPTS EXHAUSTED | Upload #%d | TransactionID: %s | Retries: %d | Last Error: %v\n",
		timestamp, uploadNum, c.transactionID, maxRetries, lastErr)

	if _, writeErr := c.errorLogFile.WriteString(logEntry); writeErr != nil {
		fmt.Printf("  \033[33m⚠\033[0m Failed to write to error log: %v\n", writeErr)
	}

	// Also print to console
	fmt.Printf("  \033[31m🔄\033[0m All %d fetch attempts exhausted for TransactionID: %s\n", maxRetries, c.transactionID)
}

func (c *Client) tokenRefreshWorker(stopChan chan bool) {
	for {
		// Calculate time until token expires with a 2-minute safety buffer
		timeUntilExpiry := time.Until(c.tokenExpiresAt)
		refreshTime := timeUntilExpiry - (2 * time.Minute)

		// If token is already expired or will expire very soon, refresh immediately
		if refreshTime <= 0 {
			refreshTime = 1 * time.Second
		}

		select {
		case <-time.After(refreshTime):
			fmt.Printf("\n\033[33m🔄 Refreshing JWT token (expires at %s)...\033[0m\n",
				c.tokenExpiresAt.Format("15:04:05"))

			if err := c.getAccessToken(); err != nil {
				fmt.Printf("\033[31m✗ Token refresh failed: %v\033[0m\n", err)
				fmt.Printf("\033[33m⚠ Will retry in 30 seconds...\033[0m\n")
				// Retry after a short delay
				time.Sleep(30 * time.Second)
			} else {
				fmt.Printf("\033[32m✓ JWT token refreshed successfully\033[0m\n")
			}

		case <-stopChan:
			fmt.Println("\033[33mStopping token refresh worker\033[0m")
			return
		}
	}
}

func (c *Client) printStormStats() {
	elapsed := time.Since(c.stats.StartTime)
	successRate := 0.0
	if c.stats.TotalUploads > 0 {
		successRate = float64(c.stats.SuccessfulUploads) / float64(c.stats.TotalUploads) * 100
	}

	fmt.Println("\n\033[36m╔════════════════════════════════════════╗\033[0m")
	fmt.Println("\033[36m║          STORM STATISTICS              ║\033[0m")
	fmt.Println("\033[36m╠════════════════════════════════════════╣\033[0m")
	fmt.Printf("\033[36m║\033[0m Total Uploads:      %-17d\033[36m║\033[0m\n", c.stats.TotalUploads)
	fmt.Printf("\033[36m║\033[0m \033[32mSuccessful:\033[0m         %-17d\033[36m║\033[0m\n", c.stats.SuccessfulUploads)
	fmt.Printf("\033[36m║\033[0m \033[31mFailed:\033[0m             %-17d\033[36m║\033[0m\n", c.stats.FailedUploads)
	fmt.Printf("\033[36m║\033[0m Total Retries:      %-17d\033[36m║\033[0m\n", c.stats.TotalRetries)
	fmt.Printf("\033[36m║\033[0m Success Rate:       %-16.2f%%\033[36m║\033[0m\n", successRate)
	fmt.Printf("\033[36m║\033[0m Elapsed Time:       %-17s\033[36m║\033[0m\n", elapsed.Round(time.Second))
	fmt.Println("\033[36m╚════════════════════════════════════════╝\033[0m")
}
