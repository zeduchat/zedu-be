package test_tokens

// func TestMessage(t *testing.T) {
// 	logger := tst.Setup()
// 	gin.SetMode(gin.TestMode)

// 	validatorRef := validator.New()
// 	db := storage.Connection()
// 	currUUID := utility.GenerateUUID()
// 	userSignUpData := models.CreateUserRequestModel{
// 		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
// 		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
// 		FirstName:   "test",
// 		LastName:    "user",
// 		Password:    "password",
// 		UserName:    fmt.Sprintf("test_username%v", currUUID),
// 	}

// 	loginData := models.LoginRequestModel{
// 		Email:    userSignUpData.Email,
// 		Password: userSignUpData.Password,
// 	}

// 	auth := auth.Controller{Db: db, Validator: validatorRef, Logger: logger,
// 		ExtReq: request.ExternalRequest{
// 			Logger: logger,
// 			Test:   true,
// 		}}
// 	r := gin.Default()

// 	tst.SignupUser(t, r, auth, userSignUpData, false)

// 	channel := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
// 	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

// 	token := tst.GetLoginToken(t, r, auth, loginData)

// 	createOrgData := models.CreateOrgRequestModel{
// 		Name:        fmt.Sprintf("TestTeam%s", currUUID),
// 		Description: "Some Random description",
// 		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
// 		Type:        "type1",
// 		Location:    "wakanda",
// 		Country:     "wakanda",
// 	}

// 	orgId, _, _ := tst.CreateOrganisation(t, r, db, org, createOrgData, token)

// 	createChannelsData := models.CreateChannelsRequest{
// 		Name:           fmt.Sprintf("TestChannels%s", utility.GenerateUUID()),
// 		Username:       fmt.Sprintf("Mr%sChannels", utility.GenerateUUID()),
// 		OrganisationID: orgId,
// 		Description:    "Some Random description",
// 	}

// 	channelId, _ := tst.CreateChannels(t, r, channel, db, createChannelsData, token)

// 	threads1 := models.Threads{
// 		ID:         utility.GenerateUUID(),
// 		ChannelsID: channelId,
// 		Status:     "pending",
// 	}
// 	db.Postgresql.Create(&threads1)

// 	tests := []struct {
// 		Name         string
// 		RequestBody  models.CreateMessageRequest
// 		ExpectedCode int
// 		Message      string
// 		Method       string
// 		Headers      map[string]string
// 		RequestURI   url.URL
// 	}{
// 		{
// 			Name:         "Successfully Get messages in a channel",
// 			RequestBody:  models.CreateMessageRequest{},
// 			ExpectedCode: http.StatusOK,
// 			Message:      "channel messages fetched successfully",
// 			Method:       http.MethodGet,
// 			RequestURI:   url.URL{Path: fmt.Sprintf("/api/v1/channels/%s/messages", channelId)},
// 			Headers: map[string]string{
// 				"Content-Type":  "application/json",
// 				"Authorization": "Bearer " + token,
// 			},
// 		},
// 	}

// 	defer func() {
// 		err := tydb.DeleteCollection(db.TypeSense, channelId)
// 		if err != nil {
// 			t.Fatalf("failed to delete collection: %v", err)
// 		}
// 		fmt.Printf("deleted collection: %v", channelId)
// 	}()

// 	for _, test := range tests {
// 		r := gin.Default()

// 		tknUrl := r.Group(fmt.Sprintf("%v", "/api/v1/channels"), middleware.Authorize(db.Postgresql))
// 		{
// 			tknUrl.GET("/:channelId/messages", channel.GetChannelsMsg)

// 		}

// 		t.Run(test.Name, func(t *testing.T) {
// 			var b bytes.Buffer
// 			json.NewEncoder(&b).Encode(test.RequestBody)

// 			req, err := http.NewRequest(test.Method, test.RequestURI.String(), &b)
// 			if err != nil {
// 				t.Fatal(err)
// 			}

// 			for i, v := range test.Headers {
// 				req.Header.Set(i, v)
// 			}

// 			rr := httptest.NewRecorder()
// 			r.ServeHTTP(rr, req)

// 			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)

// 			data := tst.ParseResponse(rr)

// 			code := int(data["status_code"].(float64))
// 			tst.AssertStatusCode(t, code, test.ExpectedCode)

// 			if test.Message != "" {
// 				message := data["message"]
// 				if message != nil {
// 					tst.AssertResponseMessage(t, message.(string), test.Message)
// 				} else {
// 					tst.AssertResponseMessage(t, "", test.Message)
// 				}

// 			}

// 		})

// 	}

// }
