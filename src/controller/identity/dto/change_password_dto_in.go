package dto

type ChangePasswordDtoIn struct {
	CurrentPassword string `json:"currentPassword,omitempty"`
	NewPassword     string `json:"newPassword"`
}
