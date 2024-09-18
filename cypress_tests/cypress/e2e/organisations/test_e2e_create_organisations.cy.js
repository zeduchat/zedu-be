describe("Create organisations API Tests", () => {
  const baseUrl = Cypress.env("baseURL");
  let authToken;

  //Authentication runs before running the tests
  before(() => {
    const email = `firsttype${Date.now()}@email.com`;
    // Sign up user
    cy.request({
      method: "POST",
      url: `${baseUrl}/auth/register`,
      body: {
        username: "Username",
        email: `firsttype${Date.now()}@email.com`,
        password: "Password123*",
        first_name: "First Name",
        last_name: "First Name",
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
        email: email,
        password: "Password123*",
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

  const email = `org${Date.now()}@email.com`;
  it("should add a new organisation successfully", () => {
    const newOrganisation = {
      name: `Organisation1${Date.now()}`,
      description: "Organisation1 Desc",
      email: email,
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };

    // Make a POST request to the API endpoint
    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: newOrganisation,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      // Assert that the response status code is 201 (Created)
      expect(response.status).to.eq(201);

      // Assert that the response body contains the organisation data
      expect(response.body.data).to.have.property(
        "name",
        newOrganisation.name.toLowerCase()
      );
    });
  });

  it("should return an error if required fields are invalid", () => {
    const incompleteOrganisation = {
      name: `Organisation1${Date.now()}`,
      description: "Organisation1 Desc",
      email: "email.com",
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };

    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: incompleteOrganisation,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
      failOnStatusCode: false, // To handle the error response gracefully
    }).then((response) => {
      // Assert that the response status code is 422 (Validation failed)
      expect(response.status).to.eq(422);
    });
  });

  it("should return an error if unauthorized user", () => {
    const incompleteOrganisation = {
      name: `Organisation1${Date.now()}`,
      description: "Organisation1 Desc",
      email: email,
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };

    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: incompleteOrganisation,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer`,
      },
      failOnStatusCode: false, // To handle the error response gracefully
    }).then((response) => {
      // Assert that the response status code is 401 (Un Authorized)
      expect(response.status).to.eq(401);
    });
  });
});
