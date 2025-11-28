# Postman Testing Guide for Huddles API

## 📦 What You Need

1. **Postman Desktop App** (Free)
   - Download from: https://www.postman.com/downloads/
   - Install and open the application

2. **Running Server**
   ```bash
   cd /Users/aguwa/Developer/HNG/telex_be
   go run main.go
   ```
   Server should be running on `http://localhost:8019`

## 🚀 Setup Instructions

### Step 1: Import Collection

1. Open Postman Desktop App
2. Click **"Import"** button (top left)
3. Select **"File"** tab
4. Navigate to: `/Users/aguwa/Developer/HNG/telex_be/postman/`
5. Select: `Huddles_API_Tests.postman_collection.json`
6. Click **"Import"**

### Step 2: Import Environment

1. Click the **"Environments"** icon (left sidebar, looks like an eye 👁️)
2. Click **"Import"** button
3. Navigate to: `/Users/aguwa/Developer/HNG/telex_be/postman/`
4. Select: `Huddles_Local_Development.postman_environment.json`
5. Click **"Import"**

### Step 3: Activate Environment

1. Click the environment dropdown (top right corner)
2. Select **"Huddles - Local Development"**
3. You should see a checkmark next to it

## 🧪 Running the Tests

### Method 1: Run All Tests (Collection Runner)

1. Click on **"Collections"** in the left sidebar
2. Click on **"Huddles API Tests"** collection
3. Click **"Run"** button (right side)
4. Click the blue **"Run Huddles API Tests"** button
5. Watch all tests execute in sequence

**What Happens:**
- ✅ Test 01: Registers a new user
- ✅ Test 02: Logs in and saves the JWT token automatically
- ✅ Test 03: Creates an organisation
- ✅ Test 04: Creates a channel
- ✅ Tests 05-10: Test huddle creation with various scenarios

### Method 2: Run Individual Tests

1. Click on **"Collections"** in the left sidebar
2. Expand **"Huddles API Tests"**
3. Click on any test (e.g., "01. Register User")
4. Click the blue **"Send"** button
5. View response in the bottom panel

**Recommended Order for Individual Testing:**
1. Run "01. Register User" first (only needed once)
2. Run "02. Login User" to get auth token
3. Run "03. Create Organisation"
4. Run "04. Create Channel"
5. Now run any of tests 05-10

## 📊 What Each Test Does

### Setup Tests (01-04)

| Test | Endpoint | Purpose | Auto-Saves |
|------|----------|---------|------------|
| 01. Register User | `POST /auth/register` | Creates test user account | `user_id` |
| 02. Login User | `POST /auth/login` | Gets JWT token | `access_token`, `user_id` |
| 03. Create Organisation | `POST /organisations` | Creates test org | `org_id` |
| 04. Create Channel | `POST /channels` | Creates test channel | `channel_id` |

### Huddle Tests (05-10)

| Test | Purpose | Expected Result |
|------|---------|-----------------|
| 05. Valid Request | Create huddle successfully | 201 Created |
| 06. With Participants | Create huddle with multiple users | 201 Created |
| 07. Missing Channel ID | Test validation | 422 Validation Error |
| 08. Invalid UUID | Test UUID format validation | 422 Validation Error |
| 09. Non-existent Channel | Test with fake channel | 404 Not Found |
| 10. Unauthorized | Test without token | 401 Unauthorized |

## 🔍 Viewing Test Results

### In Collection Runner:
- ✅ Green checkmark = Test passed
- ❌ Red X = Test failed
- Click on any test to see detailed results
- View response body, headers, and test scripts

### In Individual Request:
- **Body** tab: See the JSON response
- **Test Results** tab: See which assertions passed/failed
- **Console** (bottom): See logged values

## 📝 Understanding the Responses

### Success Response (201 Created)
```json
{
  "status": "success",
  "message": "huddle created successfully",
  "data": {
    "huddle_id": "123e4567-e89b-12d3-a456-426614174000",
    "host_id": "987fcdeb-51a2-43c1-9d4e-8a7b6c5d4e3f",
    "channel_id": "456e7890-1b2c-3d4e-5f6a-7b8c9d0e1f2a",
    "status": "active",
    "created_at": "2025-11-18T21:00:00Z",
    "started_at": "2025-11-18T21:00:00Z",
    "participants": [
      "987fcdeb-51a2-43c1-9d4e-8a7b6c5d4e3f"
    ]
  }
}
```

### Error Response (4xx)
```json
{
  "status": "error",
  "message": "channel does not exist",
  "data": null
}
```

## 🔧 Troubleshooting

### Problem: "Could not send request"
**Solution:** Make sure the Go server is running
```bash
go run main.go
```

### Problem: "401 Unauthorized" on tests 05-10
**Solution:** 
1. Run test "02. Login User" first
2. Check that the environment has the `access_token` saved
3. View environment variables: Click eye icon 👁️ (top right) → See current values

### Problem: "404 Channel not found"
**Solution:**
1. Run tests 01-04 in order first
2. Check environment variables have `channel_id` saved

### Problem: "User already exists" on Register
**Solution:**
1. This is normal if you've run the test before
2. Just skip to "02. Login User"
3. Or change the email in test 01 to something else

### Problem: Can't see environment variables
**Solution:**
1. Click the eye icon 👁️ (top right corner)
2. Select "Huddles - Local Development"
3. You'll see all saved variables

## 🎯 Pro Tips

1. **View Environment Variables:**
   - Click the eye icon 👁️ (top right)
   - You'll see all auto-saved values (token, IDs, etc.)

2. **Check Console Logs:**
   - Open Postman Console (bottom left, or View → Show Postman Console)
   - See logged values from test scripts

3. **Run Only Failed Tests:**
   - In Collection Runner, click "Run failed tests"

4. **Export Results:**
   - After running collection, click "Export Results"
   - Save as JSON or HTML

5. **Duplicate Tests:**
   - Right-click any test → "Duplicate"
   - Modify for custom scenarios

## 📂 Files Provided

1. **Huddles_API_Tests.postman_collection.json**
   - The complete test suite with 10 tests
   - Includes automatic token/ID capture
   - Test assertions built-in

2. **Huddles_Local_Development.postman_environment.json**
   - Pre-configured environment variables
   - Points to localhost:8019
   - Variables auto-populated by tests

## 🎬 Quick Start Checklist

- [ ] Install Postman Desktop
- [ ] Start Go server (`go run main.go`)
- [ ] Import collection file
- [ ] Import environment file
- [ ] Select "Huddles - Local Development" environment
- [ ] Click "Run" on collection
- [ ] Watch tests execute automatically! 🎉

## 📸 Visual Guide

### Where to find Import:
```
Postman Window
├── Top Menu Bar
│   └── Import Button ← Click here
└── Left Sidebar
    ├── Collections
    └── Environments ← Click here after importing
```

### Where to select Environment:
```
Top Right Corner
└── Dropdown menu showing "No Environment"
    └── Click to select "Huddles - Local Development"
```

### Where to run Collection:
```
Left Sidebar
└── Collections
    └── Huddles API Tests
        └── Click on collection name
            └── Click "Run" button on right side
```

## 🆘 Need Help?

- Check server logs in terminal
- Open Postman Console (View → Show Postman Console)
- Check environment variables (eye icon 👁️)
- Ensure server is running on port 8019
- Make sure database is accessible

---

**Happy Testing! 🚀**
