package types

import (
	"time"

	"github.com/go-dev-frame/sponge/pkg/sgorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.


// CreateUsersRequest request params
type CreateUsersRequest struct {
	Email  string `json:"email" binding:""`
	EncryptedPassword  string `json:"encryptedPassword" binding:""`
	ResetPasswordToken  string `json:"resetPasswordToken" binding:""`
	ResetPasswordSentAt  *time.Time `json:"resetPasswordSentAt" binding:""`
	RememberCreatedAt  *time.Time `json:"rememberCreatedAt" binding:""`
	SignInCount  int `json:"signInCount" binding:""`
	CurrentSignInAt  *time.Time `json:"currentSignInAt" binding:""`
	LastSignInAt  *time.Time `json:"lastSignInAt" binding:""`
	CurrentSignInIP  string `json:"currentSignInIP" binding:""`
	LastSignInIP  string `json:"lastSignInIP" binding:""`
	ConfirmationToken  string `json:"confirmationToken" binding:""`
	ConfirmedAt  *time.Time `json:"confirmedAt" binding:""`
	ConfirmationSentAt  *time.Time `json:"confirmationSentAt" binding:""`
	UnconfirmedEmail  string `json:"unconfirmedEmail" binding:""`
	FailedAttempts  int `json:"failedAttempts" binding:""`
	UnlockToken  string `json:"unlockToken" binding:""`
	LockedAt  *time.Time `json:"lockedAt" binding:""`
	Admin  int `json:"admin" binding:""`
	Username  string `json:"username" binding:""`
	RememberToken  string `json:"rememberToken" binding:""`
}

// UpdateUsersByIDRequest request params
type UpdateUsersByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id

	Email  string `json:"email" binding:""`
	EncryptedPassword  string `json:"encryptedPassword" binding:""`
	ResetPasswordToken  string `json:"resetPasswordToken" binding:""`
	ResetPasswordSentAt  *time.Time `json:"resetPasswordSentAt" binding:""`
	RememberCreatedAt  *time.Time `json:"rememberCreatedAt" binding:""`
	SignInCount  int `json:"signInCount" binding:""`
	CurrentSignInAt  *time.Time `json:"currentSignInAt" binding:""`
	LastSignInAt  *time.Time `json:"lastSignInAt" binding:""`
	CurrentSignInIP  string `json:"currentSignInIP" binding:""`
	LastSignInIP  string `json:"lastSignInIP" binding:""`
	ConfirmationToken  string `json:"confirmationToken" binding:""`
	ConfirmedAt  *time.Time `json:"confirmedAt" binding:""`
	ConfirmationSentAt  *time.Time `json:"confirmationSentAt" binding:""`
	UnconfirmedEmail  string `json:"unconfirmedEmail" binding:""`
	FailedAttempts  int `json:"failedAttempts" binding:""`
	UnlockToken  string `json:"unlockToken" binding:""`
	LockedAt  *time.Time `json:"lockedAt" binding:""`
	Admin  int `json:"admin" binding:""`
	Username  string `json:"username" binding:""`
	RememberToken  string `json:"rememberToken" binding:""`
}

// UsersObjDetail detail
type UsersObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id

	Email  string `json:"email"`
	EncryptedPassword  string `json:"encryptedPassword"`
	ResetPasswordToken  string `json:"resetPasswordToken"`
	ResetPasswordSentAt  *time.Time `json:"resetPasswordSentAt"`
	RememberCreatedAt  *time.Time `json:"rememberCreatedAt"`
	SignInCount  int `json:"signInCount"`
	CurrentSignInAt  *time.Time `json:"currentSignInAt"`
	LastSignInAt  *time.Time `json:"lastSignInAt"`
	CurrentSignInIP  string `json:"currentSignInIP"`
	LastSignInIP  string `json:"lastSignInIP"`
	ConfirmationToken  string `json:"confirmationToken"`
	ConfirmedAt  *time.Time `json:"confirmedAt"`
	ConfirmationSentAt  *time.Time `json:"confirmationSentAt"`
	UnconfirmedEmail  string `json:"unconfirmedEmail"`
	FailedAttempts  int `json:"failedAttempts"`
	UnlockToken  string `json:"unlockToken"`
	LockedAt  *time.Time `json:"lockedAt"`
	CreatedAt  *time.Time `json:"createdAt"`
	UpdatedAt  *time.Time `json:"updatedAt"`
	Admin  int `json:"admin"`
	Username  string `json:"username"`
	RememberToken  string `json:"rememberToken"`
}


// CreateUsersReply only for api docs
type CreateUsersReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// UpdateUsersByIDReply only for api docs
type UpdateUsersByIDReply struct {
	Code int      `json:"code"` // return code
	Msg  string   `json:"msg"`  // return information description
	Data struct{} `json:"data"` // return data
}

// GetUsersByIDReply only for api docs
type GetUsersByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Users UsersObjDetail `json:"users"`
	} `json:"data"` // return data
}

// DeleteUsersByIDReply only for api docs
type DeleteUsersByIDReply struct {
	Code int      `json:"code"` // return code
	Msg  string   `json:"msg"`  // return information description
	Data struct{} `json:"data"` // return data
}

// DeleteUserssByIDsReply only for api docs
type DeleteUserssByIDsReply struct {
	Code int      `json:"code"` // return code
	Msg  string   `json:"msg"`  // return information description
	Data struct{} `json:"data"` // return data
}

// ListUserssRequest request params
type ListUserssRequest struct {
	query.Params
}

// ListUserssReply only for api docs
type ListUserssReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Userss []UsersObjDetail `json:"userss"`
	} `json:"data"` // return data
}

// DeleteUserssByIDsRequest request params
type DeleteUserssByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// GetUsersByConditionRequest request params
type GetUsersByConditionRequest struct {
	query.Conditions
}

// GetUsersByConditionReply only for api docs
type GetUsersByConditionReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Users UsersObjDetail `json:"users"`
	} `json:"data"` // return data
}

// ListUserssByIDsRequest request params
type ListUserssByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// ListUserssByIDsReply only for api docs
type ListUserssByIDsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Userss []UsersObjDetail `json:"userss"`
	} `json:"data"` // return data
}
