describe("Retrieve organisations API Tests", () => {
  const baseUrl = Cypress.env("baseURL");
  let authToken;
  let organisationId;
  const orgEmail = `org${Date.now()}@email.com`;

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

  it("should retrieve organisation successfully", () => {
    let createData = {
      name: "Organisation1",
      description: "Organisation1 Desc",
      email: orgEmail,
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };

    // Make a POST request to the API endpoint
    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: createData,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      // Assert that the response status code is 201 (Created)
      expect(response.status).to.eq(201);
      organisationId = response.body.data.id;

      cy.request({
        method: "GET",
        url: `${baseUrl}/organisations/${organisationId}`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}`,
        },
      }).then((response) => {
        // Assert that the response status code is 200 (OK)
        expect(response.status).to.eq(200);

        // Assert that the response body contains the organisation data
        expect(response.body.data).to.have.property(
          "name",
          createData.name.toLowerCase()
        );
      });
    });
  });

  it("should return an error if unauthorized user", () => {
    let createData = {
      name: "Organisation1",
      description: "Organisation1 Desc",
      email: orgEmail,
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };

    // Make a POST request to the API endpoint
    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: createData,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      // Assert that the response status code is 201 (Created)
      expect(response.status).to.eq(201);
      organisationId = response.body.data.id;

      cy.request({
        method: "GET",
        url: `${baseUrl}/organisations/${organisationId}`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer`,
        },
        failOnStatusCode: false,
      }).then((response) => {
        // Assert that the response status code is 401 (Unauthorized)
        expect(response.status).to.eq(401);
      });
    });
  });

  it("should return an error if organisation id is invalid", () => {
    let createData = {
      name: "Organisation1",
      description: "Organisation1 Desc",
      email: orgEmail,
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };

    // Make a POST request to the API endpoint
    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: createData,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      // Assert that the response status code is 201 (Created)
      expect(response.status).to.eq(201);
      organisationId = response.body.data.id;

      cy.request({
        method: "GET",
        url: `${baseUrl}/organisations/`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}`,
        },
        failOnStatusCode: false,
      }).then((response) => {
        // Assert that the response status code is 404 (Not found)
        expect(response.status).to.eq(404);
      });
    });
  });

  it("should return an error if when data is incorrect", () => {
    let createData = {
      name: "Organisation1",
      description: "Organisation1 Desc",
      email: orgEmail,
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };

    // Make a POST request to the API endpoint
    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: createData,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      // Assert that the response status code is 201 (Created)
      expect(response.status).to.eq(201);
      organisationId = response.body.data.id;

      cy.request({
        method: "GET",
        url: `${baseUrl}/organisations/${organisationId}%`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}`,
        },
        failOnStatusCode: false,
      }).then((response) => {
        // Assert that the response status code is 400 (Bad request error)
        expect(response.status).to.eq(400);
      });
    });
  });
});
