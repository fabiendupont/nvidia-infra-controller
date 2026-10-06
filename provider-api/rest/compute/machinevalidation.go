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

package compute

import (
	"time"

	"github.com/NVIDIA/infra-controller/provider-api/rest"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	validationErrorStringLength64 = "must be at least 2 characters and maximum 64 characters"
)

// APIMachineValidationTest data structure to capture MachineValidation test
type APIMachineValidationTest struct {
	// TestID id of the test
	TestID string `json:"testID"`
	// Name test name
	Name string `json:"name"`
	// Description test description
	Description string `json:"description"`
	// Contexts list of test contexts
	Contexts []string `json:"contexts"`
	// ContainerImgName test container image name
	ContainerImgName string `json:"containerImgName"`
	// IsExecuteInHost indicates to run test command using chroot in case of container
	IsExecuteInHost bool `json:"isExecuteInHost"`
	// ContainerArgs test container arguments
	ContainerArgs string `json:"containerArgs"`
	// Command test command
	Command string `json:"command"`
	// Args test arguments
	Args string `json:"args"`
	// ExtraErrFile test command error output file
	ExtraErrFile string `json:"extraErrFile"`
	// ExternalConfigFile test external configuration file
	ExternalConfigFile string `json:"externalConfigFile"`
	// PreCondition test pre-condition
	PreCondition string `json:"preCondition"`
	// Timeout test timeout in seconds (default 7200)
	Timeout int64 `json:"timeout"`
	// ExtraOutputFile test command standard output file
	ExtraOutputFile string `json:"extraOutputFile"`
	// Version test version
	Version string `json:"version"`
	// SupportedPlatforms list of supported platform for a test
	SupportedPlatforms []string `json:"supportedPlatforms"`
	// ModifiedBy user that last modified test
	ModifiedBy string `json:"modifiedBy"`
	// IsVerified indicates if test verified or not
	IsVerified bool `json:"isVerified"`
	// IsReadOnly indicates if the test is read-only or not
	IsReadOnly bool `json:"isReadOnly"`
	// CustomTags list of custom tags for test
	CustomTags []string `json:"customTags"`
	// Components list of system components for test
	Components []string `json:"components"`
	// LastModifiedAt last time test modified
	LastModifiedAt string `json:"lastModifiedAt"`
	// IsEnabled indicates if test is enabled or not
	IsEnabled bool `json:"isEnabled"`
}

// APIMachineValidationTestCreateRequest is the data structure to capture user request to create a machine validation test
type APIMachineValidationTestCreateRequest struct {
	// Name test name
	Name string `json:"name"`
	// Command test command
	Command string `json:"command"`
	// Args test command arguments
	Args string `json:"args"`
	// Description test description
	Description *string `json:"description"`
	// Contexts list of contexts for a test
	Contexts []string `json:"contexts"`
	// ContainerImgName test container image name
	ContainerImgName *string `json:"containerImgName"`
	// IsExecuteInHost run test command using chroot in case of container
	IsExecuteInHost *bool `json:"isExecuteInHost"`
	// ContainerArgs test container arguments
	ContainerArgs *string `json:"containerArgs"`
	// ExtraErrFile test command error output file
	ExtraErrFile *string `json:"extraErrFile"`
	// ExternalConfigFile test external configuration file
	ExternalConfigFile *string `json:"externalConfigFile"`
	// PreCondition test pre-condition
	PreCondition *string `json:"preCondition"`
	// Timeout test command timeout in seconds (default 7200)
	Timeout *int64 `json:"timeout"`
	// ExtraOutputFile test command standard output file
	ExtraOutputFile *string `json:"extraOutputFile"`
	// SupportedPlatforms list of supported platform for a test
	SupportedPlatforms []string `json:"supportedPlatforms"`
	// IsReadOnly indicates if test is read-only or not
	IsReadOnly *bool `json:"isReadOnly"`
	// CustomTags list of custom tags for a test
	CustomTags []string `json:"customTags"`
	// Components list of system components for a test
	Components []string `json:"components"`
	// IsEnabled indicates if test is enabled or not
	IsEnabled *bool `json:"isEnabled"`
}

// Validate ensures that the values passed in request are acceptable
func (req APIMachineValidationTestCreateRequest) Validate() error {
	err := validation.ValidateStruct(&req,
		validation.Field(&req.Name,
			validation.Required.Error(validationErrorStringLength64),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 64).Error(validationErrorStringLength64)),
		validation.Field(&req.Command,
			validation.Required.Error(rest.ValidationErrorStringLength),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 256).Error(rest.ValidationErrorStringLength)),
		validation.Field(&req.Args,
			validation.Required.Error(rest.ValidationErrorValueRequired),
			validation.By(rest.ValidateNameCharacters)),
	)
	if err != nil {
		return err
	}
	return nil
}

// APIMachineValidationTestUpdateRequest is the data structure to capture user request to update a machine validation test
type APIMachineValidationTestUpdateRequest struct {
	// Name test name
	Name *string `json:"name"`
	// Command test command
	Command *string `json:"command"`
	// Args test command arguments
	Args *string `json:"args"`
	// Description test description
	Description *string `json:"description"`
	// Contexts list of contexts for a test
	Contexts []string `json:"contexts"`
	// ContainerImgName test container image name
	ContainerImgName *string `json:"containerImgName"`
	// IsExecuteInHost run test command using chroot in case of container
	IsExecuteInHost *bool `json:"isExecuteInHost"`
	// ContainerArgs test container arguments
	ContainerArgs *string `json:"containerArgs"`
	// ExtraErrFile test command error output file
	ExtraErrFile *string `json:"extraErrFile"`
	// ExternalConfigFile test external configuration file
	ExternalConfigFile *string `json:"externalConfigFile"`
	// PreCondition test pre-condition
	PreCondition *string `json:"preCondition"`
	// Timeout test command timeout in seconds (default 7200)
	Timeout *int64 `json:"timeout"`
	// ExtraOutputFile test command standard output file
	ExtraOutputFile *string `json:"extraOutputFile"`
	// SupportedPlatforms list of supported platform for a test
	SupportedPlatforms []string `json:"supportedPlatforms"`
	// IsVerified indicates if test verified or not
	IsVerified *bool `json:"isVerified"`
	// CustomTags list of custom tags for a test
	CustomTags []string `json:"customTags"`
	// Components list of system components for a test
	Components []string `json:"components"`
	// IsEnabled indicates if test is enabled or not
	IsEnabled *bool `json:"isEnabled"`
}

// APIMachineValidationTestsFilter is the data structure for filtering machine validation tests
type APIMachineValidationTestsFilter struct {
	// SupportedPlatforms list of supported platform for a test
	SupportedPlatforms []string `query:"supportedPlatforms"`
	// Contexts list of contexts for a test
	Contexts []string `query:"contexts"`
	// IsReadOnly indicates if test is read-only or not
	IsReadOnly *bool `query:"isReadOnly"`
	// CustomTags list of custom tags for a test
	CustomTags []string `query:"customTags"`
	// IsEnabled indicates if test is enabled or not
	IsEnabled *bool `query:"isEnabled"`
	// IsVerified indicates if test verified or not
	IsVerified *bool `query:"isVerified"`
}

// APIMachineValidationResult is the data structure for a machine validation result
type APIMachineValidationResult struct {
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Command      string    `json:"command"`
	Args         string    `json:"args"`
	StdOut       string    `json:"stdOut"`
	StdErr       string    `json:"stdErr"`
	Context      string    `json:"context"`
	ExitCode     int       `json:"exitCode"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime"`
	ValidationID string    `json:"validationID"`
	TestID       string    `json:"testID"`
}

// APIMachineValidationState represents the state of a machine validation
type APIMachineValidationState string

const (
	MachineValidationStarted    APIMachineValidationState = "Started"
	MachineValidationInProgress APIMachineValidationState = "InProgress"
	MachineValidationSuccess    APIMachineValidationState = "Success"
	MachineValidationFailed     APIMachineValidationState = "Failed"
	MachineValidationSkipped    APIMachineValidationState = "Skipped"
)

// APIMachineValidationStatus is the data structure for a machine validation status
type APIMachineValidationStatus struct {
	State     APIMachineValidationState `json:"state"`
	Total     int                       `json:"total"`
	Completed int                       `json:"completed"`
}

// APIMachineValidationRun is the data structure for a machine validation run
type APIMachineValidationRun struct {
	ValidationID           string                     `json:"validationID"`
	MachineID              string                     `json:"machineID"`
	StartTime              time.Time                  `json:"startTime"`
	EndTime                *time.Time                 `json:"endTime"`
	Name                   string                     `json:"name"`
	Context                string                     `json:"context"`
	Status                 APIMachineValidationStatus `json:"status"`
	DurationToCompleteSecs int                        `json:"durationToCompleteSecs"`
}

// APIMachineValidationExternalConfig is the data structure for a machine validation external config
type APIMachineValidationExternalConfig struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Config      []byte    `json:"config"`
	Version     string    `json:"version"`
	Timestamp   time.Time `json:"timestamp"`
}

// APIMachineValidationExternalConfigCreateRequest is the data structure for creating a machine validation external config
type APIMachineValidationExternalConfigCreateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Config      []byte  `json:"config"`
}

// Validate ensures that the values passed in request are acceptable
func (req APIMachineValidationExternalConfigCreateRequest) Validate() error {
	err := validation.ValidateStruct(&req,
		validation.Field(&req.Name,
			validation.Required.Error(validationErrorStringLength64),
			validation.By(rest.ValidateNameCharacters),
			validation.Length(2, 64).Error(validationErrorStringLength64)),
		validation.Field(&req.Config,
			validation.Required.Error(rest.ValidationErrorValueRequired)),
	)
	if err != nil {
		return err
	}
	return nil
}

// APIMachineValidationExternalConfigUpdateRequest is the data structure for updating a machine validation external config
type APIMachineValidationExternalConfigUpdateRequest struct {
	Description *string `json:"description"`
	Config      []byte  `json:"config"`
}
