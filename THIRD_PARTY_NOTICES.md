# Third-Party Notices

This file records the runtime dependencies added for embedded public access. Versions are resolved
from the committed Go module graph; checksums are recorded in `go.sum`. The packaged application
does not download these modules or a provider executable at runtime. The canonical package graph
installs this reviewed file unchanged at `Contents/Resources/THIRD_PARTY_NOTICES.md`.

## Reviewed runtime module inventory

- github.com/jpillora/backoff@v1.0.0 — MIT; Copyright (c) 2017 Jaime Pillora
- github.com/keybase/go-keychain@v0.0.1 — MIT; Copyright (c) 2015 Keybase
- go.uber.org/multierr@v1.11.0 — MIT; Copyright (c) 2017-2021 Uber Technologies, Inc.
- golang.ngrok.com/muxado/v2@v2.0.1 — MIT; Copyright 2023 ngrok, Inc.
- golang.ngrok.com/ngrok/v2@v2.1.4 — MIT; Copyright 2022 ngrok, Inc.
- golang.org/x/net@v0.56.0 — BSD-3-Clause; Copyright 2009 The Go Authors
- google.golang.org/protobuf@v1.36.11 — BSD-3-Clause; Copyright (c) 2018 The Go Authors

## ngrok-go

`golang.ngrok.com/ngrok/v2` and `golang.ngrok.com/muxado/v2` are distributed under the MIT License.
Their copyright notices are listed in the reviewed inventory above.

## go-keychain

`github.com/keybase/go-keychain` is distributed under the MIT License. Its copyright notice is
listed in the reviewed inventory above.

## MIT License terms

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
associated documentation files (the "Software"), to deal in the Software without restriction,
including without limitation the rights to use, copy, modify, merge, publish, distribute,
sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The copyright notice applicable to each module and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT
OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

## BSD-3-Clause License terms

Redistribution and use in source and binary forms, with or without modification, are permitted
provided that the following conditions are met:

1. Redistributions of source code must retain the applicable copyright notice, this list of
   conditions and the following disclaimer.
2. Redistributions in binary form must reproduce the applicable copyright notice, this list of
   conditions and the following disclaimer in the documentation and/or other materials provided
   with the distribution.
3. Neither the name of the copyright holder nor the names of its contributors may be used to
   endorse or promote products derived from this software without specific prior written
   permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR
IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND
FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR
CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER
IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT
OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
