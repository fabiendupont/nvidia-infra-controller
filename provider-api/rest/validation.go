/*
 * SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rest

import (
	"errors"
	"fmt"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Common validation error messages
const (
	ValidationErrorValueRequired           = "a value is required"
	ValidationErrorInvalidUUID             = "must be a valid UUID"
	ValidationErrorStringLength            = "must be at least 2 characters and maximum 256 characters"
	ValidationErrorDescriptionStringLength = "maximum 1024 characters are allowed in description"
	ValidationErrorInvalidIPAddress        = "invalid IP address"
	ValidationErrorInvalidIPv4Address      = "invalid IPv4 address"
	ValidationErrorInvalidIPv6Address      = "invalid IPv6 address"
	ValidationErrorInvalidHostname         = "invalid hostname"

	ValidationCommonErrorField = "__all__"
)

var (
	LeadingWhitespaceRegexp  = regexp.MustCompile(`^\s+.*`)
	TrailingWhitespaceRegexp = regexp.MustCompile(`.*\s+$`)
	NotAllWhitespaceRegexp   = regexp.MustCompile(`[^\s]+`)

	ValidationErrorNameHasLeadingWhitespace  = errors.New("name field has leading whitespace")
	ValidationErrorNameHasTrailingWhitespace = errors.New("name field has trailing whitespace")

	// Label restrictions
	LabelKeyMaxLength   = 255
	LabelValueMaxLength = 255
	LabelCountMax       = 10

	// Label validation error messages
	ErrValidationLabelKeyEmpty    = errors.New("one or more labels do not have a key specified")
	ErrValidationLabelKeyLength   = fmt.Errorf("label key must contain at least 1 character and a maximum of %v characters", LabelKeyMaxLength)
	ErrValidationLabelValueLength = fmt.Errorf("label value cannot exceed a maximum of %v characters", LabelValueMaxLength)
	ErrValidationLabelCount       = fmt.Errorf("up to %v key/value pairs can be specified in labels", LabelCountMax)
)

// ValidateLabels validates optional API label maps (count, keys, values). Returns nil when labels is nil.
func ValidateLabels(labels map[string]string) error {
	if labels == nil {
		return nil
	}
	if len(labels) > LabelCountMax {
		return validation.Errors{
			"labels": ErrValidationLabelCount,
		}
	}

	keyErrMsg := ErrValidationLabelKeyLength.Error()
	valueErrMsg := ErrValidationLabelValueLength.Error()

	for key, value := range labels {
		if key == "" {
			return validation.Errors{
				"labels": ErrValidationLabelKeyEmpty,
			}
		}

		err := validation.Validate(key,
			validation.Match(NotAllWhitespaceRegexp).Error("label key consists only of whitespace"),
			validation.Length(1, LabelKeyMaxLength).Error(keyErrMsg),
		)
		if err != nil {
			return validation.Errors{
				"labels": err,
			}
		}

		err = validation.Validate(value,
			validation.When(value != "",
				validation.Length(0, LabelValueMaxLength).Error(valueErrMsg),
			),
		)
		if err != nil {
			return validation.Errors{
				"labels": err,
			}
		}
	}

	return nil
}

// ValidateNameCharacters is a utility function to lexically validate the name field.
// Currently checks for leading or trailing whitespaces.
// NOTE: Can only be used in conjunction with validation.Required or with
// validation.When(name != nil, validation.By(ValidateNameCharacters))
func ValidateNameCharacters(value interface{}) error {
	s, ok := value.(string)
	var name string
	if !ok {
		sPtr, ok := value.(*string)
		if !ok {
			return errors.New("value in name field must be a string type")
		}
		if sPtr == nil {
			return errors.New("name field cannot be nil")
		}
		name = *sPtr
	} else {
		name = s
	}
	if LeadingWhitespaceRegexp.Match([]byte(name)) {
		return ValidationErrorNameHasLeadingWhitespace
	}
	if TrailingWhitespaceRegexp.Match([]byte(name)) {
		return ValidationErrorNameHasTrailingWhitespace
	}
	return nil
}
