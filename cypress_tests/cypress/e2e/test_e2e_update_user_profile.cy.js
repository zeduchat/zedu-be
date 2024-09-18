describe("Update user profile", () => {
  const baseUrl = Cypress.env("baseURL");
  let authToken;

  const now = Date.now();
  const email = `my-test-user${now}@email.com`;
  const username = "specialUsername";
  const password = "password";
  const first_name = "FirstName";
  const last_name = "LastName";

  // Authentication runs before running the tests
  before(() => {
    // Sign up user
    cy.request({
      method: "POST",
      url: `${baseUrl}/auth/register`,
      body: {
        username,
        email,
        password,
        first_name,
        last_name,
      },
      headers: {
        "Content-Type": "application/json",
      },
    }).then((response) => {
      // Assert that the signup was successful
      expect(response.status).to.eq(201);
    });

    // Log in to obtain the token
    cy.request({
      method: "POST",
      url: `${baseUrl}/auth/login`,
      body: {
        email,
        password,
      },
      headers: {
        "Content-Type": "application/json",
      },
    }).then((response) => {
      // Assert that the login was successful
      expect(response.status).to.eq(200);

      // Save the token for use in subsequent requests
      authToken = response.body.data.access_token;
    });
  });

  it("Allows authenticated users to update their profile information", () => {
    const formData = new FormData();
    const full_name = "New FullName";
    const user_name = "newUsername";
    formData.append("full_name", full_name);
    formData.append("user_name", user_name);

    cy.request({
      method: "PATCH",
      url: `${baseUrl}/profile`,
      body: formData,
      headers: {
        "Content-Type": "multipart/form-data",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      // Assert that the reponse status is 200
      expect(response.status).to.eq(200);

      const text = new TextDecoder().decode(response.body);
      const jsonObject = JSON.parse(text);

      // Assert that the user profile was updated
      expect(jsonObject.message).to.eq("Profile updated successfully");
    });
  });

  it("should fail if a user is not authenticated", () => {
    const formData = new FormData();
    const full_name = "New FullName";
    const user_name = "newUsername";
    formData.append("full_name", full_name);
    formData.append("user_name", user_name);

    cy.request({
      method: "PATCH",
      url: `${baseUrl}/profile`,
      failOnStatusCode: false,
      body: formData,
      headers: {
        "Content-Type": "multipart/form-data",
      },
    }).then((response) => {
      // Assert that the reponse status is 401 because we did not send the auth token
      expect(response.status).to.eq(401);
    });
  });
});
