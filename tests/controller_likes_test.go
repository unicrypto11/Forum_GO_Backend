package tests

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLikePost(t *testing.T) {

	var firstUserEmail, secondUserEmail string
	var firstPostID uint64

	err := refreshUserPostAndLikeTable()
	if err != nil {
		log.Fatal(err)
	}
	users, posts, err := seedUsersAndPosts()
	if err != nil {
		log.Fatalf("Cannot seed user %v\n", err)
	}

	// Get only the first user
	for _, user := range users {
		if user.ID == 1 {
			firstUserEmail = user.Email
		}
		if user.ID == 2 {
			secondUserEmail = user.Email
		}
	}
	// Get only the first post, which belongs to first user
	for _, post := range posts {
		if post.ID == 2 {
			continue
		}
		firstPostID = post.ID
	}
	// Login both users
	// user 1 and user 2 password are the same, you can change if you want (Note by the time they are hashed and saved in the db, they are different)
	// Note: the value of the user password before it was hashed is "password". so:
	password := "password"

	// Login First User
	tokenInterface1, err := server.SignIn(firstUserEmail, password)
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}
	token1 := tokenInterface1["token"] //get only the token
	firstUserToken := fmt.Sprintf("Bearer %v", token1)

	// Login Second User
	tokenInterface2, err := server.SignIn(secondUserEmail, password)
	if err != nil {
		log.Fatalf("cannot login: %v\n", err)
	}
	token2 := tokenInterface2["token"] //get only the token
	secondUserToken := fmt.Sprintf("Bearer %v", token2)

	samples := []struct {
		postIDString string
		statusCode   int
		userID       uint32
		postID       uint64
		tokenGiven   string
	}{
		{
			// User 1 can like his post
			postIDString: strconv.Itoa(int(firstPostID)), //we need the id as a string
			statusCode:   201,
			userID:       1,
			postID:       firstPostID,
			tokenGiven:   firstUserToken,
		},
		{
			// User 2 can also like user 1 post
			postIDString: strconv.Itoa(int(firstPostID)),
			statusCode:   201,
			userID:       2,
			postID:       firstPostID,
			tokenGiven:   secondUserToken,
		},
		{
			// An authenticated user cannot like a post more than once
			postIDString: strconv.Itoa(int(firstPostID)),
			statusCode:   500,
			tokenGiven:   firstUserToken,
		},
		{
			// Not authenticated (No token provided)
			postIDString: strconv.Itoa(int(firstPostID)),
			statusCode:   401,
			tokenGiven:   "",
		},
		{
			// Wrong Token
			postIDString: strconv.Itoa(int(firstPostID)),
			statusCode:   401,
			tokenGiven:   "This is an incorrect token",
		},
	}

	for _, v := range samples {

		gin.SetMode(gin.TestMode)

		r := gin.Default()

		r.POST("/likes/:id", server.LikePost)
		req, err := http.NewRequest(http.MethodPost, "/likes/"+v.postIDString, nil)
		req.Header.Set("Authorization", v.tokenGiven)
		if err != nil {
			t.Errorf("this is the error: %v\n", err)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		responseInterface := make(map[string]interface{})
		err = json.Unmarshal([]byte(rr.Body.String()), &responseInterface)
		if err != nil {
			t.Errorf("Cannot convert to json: %v", err)
		}
		assert.Equal(t, rr.Code, v.statusCode)

		if v.statusCode == 201 {
			responseMap := responseInterface["response"].(map[string]interface{})
			assert.Equal(t, responseMap["post_id"], float64(v.postID))
			assert.Equal(t, responseMap["user_id"], float64(v.userID))
		}
		if v.statusCode == 401 || v.statusCode == 422 || v.statusCode == 500 {
			responseMap := responseInterface["error"].(map[string]interface{})

			if responseMap["Unauthorized"] != nil {
				assert.Equal(t, responseMap["Unauthorized"], "Unauthorized")
			}
			if responseMap["Double_like"] != nil {
				assert.Equal(t, responseMap["Double_like"], "You cannot like this post twice")
			}
		}
	}
}

func TestGetLikes(t *testing.T) {

	gin.SetMode(gin.TestMode)

	r := gin.Default()

	err := refreshUserPostAndLikeTable()
	if err != nil {
		log.Fatal(err)
	}
	post, users, likes, err := seedUsersPostsAndLikes()
	if err != nil {
		log.Fatalf("Cannot seed tables %v\n", err)
	}
	postIDString := strconv.Itoa(int(post.ID))

	r.GET("/likes/:id", server.GetLikes)
	req, err := http.NewRequest(http.MethodGet, "/likes/"+postIDString, nil)
	if err != nil {
		t.Errorf("this is the error: %v\n", err)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	likesInterface := make(map[string]interface{})
	err = json.Unmarshal([]byte(rr.Body.String()), &likesInterface)
	if err != nil {
		log.Fatalf("Cannot convert to json: %v\n", err)
	}
	theLikes := likesInterface["response"].([]interface{})
	assert.Equal(t, rr.Code, http.StatusOK)
	assert.Equal(t, len(theLikes), len(likes))
	assert.Equal(t, len(users), 2)
}
