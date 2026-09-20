package dto

type UpdateEventMeetingLinkDto struct {
	MeetingLink string `json:"meeting_link" example:"https://meet.example.com/sala" maxLength:"500"`
}
