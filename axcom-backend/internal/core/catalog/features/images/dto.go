// Copyright 2026 Axiolon Labs
// SPDX-License-Identifier: Apache-2.0

package images

type PresignImageRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType,omitempty"`
}

type PresignImagesRequest struct {
	Files []PresignImageRequest `json:"files"`
}

type PresignImageResponse struct {
	Filename  string `json:"filename"`
	UploadURL string `json:"uploadUrl"`
	PublicURL string `json:"publicUrl"`
	Key       string `json:"key"`
	Method    string `json:"method"`
}

type PresignImagesResponse struct {
	Uploads []PresignImageResponse `json:"uploads"`
}

type RegisterImageRequest struct {
	Key       string `json:"key"`
	IsPrimary bool   `json:"isPrimary,omitempty"`
}

type RegisterImagesRequest struct {
	Images []RegisterImageRequest `json:"images"`
}
