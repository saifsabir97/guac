//
// Copyright 2021 The AFF Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processor

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
)

var (
	documentProcessors = map[DocumentType]DocumentProcessor{}
)

func init() {
	// RegisterDocumentProcessor()
}

func RegisterDocumentProcessor(p DocumentProcessor, d DocumentType) {
	if _, ok := documentProcessors[d]; ok {
		logrus.Warnf("the document processor is being overwritten: %s", d)
	}
	documentProcessors[d] = p
}

func Process(i *Document) ([]*Document, error) {
	docsToUnpack := []*Document{i}
	finalDocs := []*Document{}
	for len(docsToUnpack) > 0 {
		logrus.Debugf("%v documents left in queue", len(docsToUnpack))
		dd := docsToUnpack[0]
		docsToUnpack = docsToUnpack[1:]

		ds, err := processDocument(dd)
		// TODO: return a policy type error to provide better log warnings
		if err != nil {
			continue
		}

		logrus.Debugf("unpacked document to %v documents", len(ds))
		if len(ds) > 0 {
			docsToUnpack = append(docsToUnpack, ds...)
		} else {
			dd.SourceInformation = i.SourceInformation
			finalDocs = append(finalDocs, dd)
		}
	}
	return finalDocs, nil
}

func processDocument(i *Document) ([]*Document, error) {
	if err := i.validateFormat(); err != nil {
		return nil, err
	}

	trustInfo, err := i.validateDocument()
	if err != nil {
		return nil, err
	}

	// pass trustInfo into policy
	_ = trustInfo

	ds, err := i.unpackDocument()
	if err != nil {
		return nil, fmt.Errorf("unable to unpack document: %w", err)
	}

	return ds, nil
}

func (i *Document) validateFormat() error {
	switch i.Format {
	case FormatJSON:
		if !json.Valid(i.Blob) {
			return fmt.Errorf("invalid JSON document")
		}
		break
	default:
		return fmt.Errorf("invalid document format type: %v", i.Format)
	}
	return nil
}

func (i *Document) validateDocument() (map[string]interface{}, error) {
	p, ok := documentProcessors[i.Type]
	if !ok {
		return nil, fmt.Errorf("no document processor registered for type: %s", i.Type)
	}

	if err := p.ValidateSchema(i); err != nil {
		return nil, fmt.Errorf("error validating document schema: %w", err)
	}

	trustInfo, err := p.ValidateTrustInformation(i)
	if err != nil {
		return nil, fmt.Errorf("error validating trust information: %w", err)
	}

	return trustInfo, nil
}

func (i *Document) unpackDocument() ([]*Document, error) {
	p, ok := documentProcessors[i.Type]
	if !ok {
		return nil, fmt.Errorf("no document processor registered for type: %s", i.Type)
	}
	return p.Unpack(i)
}

func (i *Document) validate() (bool, error) {
	if err := i.validateFormat(); err != nil {
		return false, err
	}

	trustInfo, err := i.validateDocument()
	if err != nil {
		return false, err
	}

	// pass trustInfo into policy
	_ = trustInfo

	return true, nil
}