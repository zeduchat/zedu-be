describe("Create organisations and roles API Tests", () => {
  const baseUrl = Cypress.env("baseURL");
  let authToken;
  let organizationId;
  before(() => {
    const email = `firsttype${Date.now()}@email.com`;
    // Sign up user
    cy.request({
      method: "POST",
      url: `${baseUrl}/auth/register`,
      body: {
        username: "Username",
        email: email,
        password: "Password123*",
        first_name: "First Name",
        last_name: "Last Name",
      },
      headers: {
        "Content-Type": "application/json",
      },
    }).then((response) => {
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
      expect(response.status).to.eq(200);
      authToken = response.body.data.access_token;
    });
  });
  it("should add a new organisation successfully", () => {
    const newOrganisation = {
      name: `Organisation2${Date.now()}`,
      description: "Organisation1 Desc",
      email: `org${Date.now()}@email.com`,
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };
    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: newOrganisation,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      expect(response.status).to.eq(201);
      expect(response.body.data).to.have.property(
        "name",
        newOrganisation.name.toLowerCase()
      );
      organizationId = response.body.data.id; // Save the organization ID for later use
    });
  });
  it("should return an error if required fields are invalid for organization creation", () => {
    const incompleteOrganisation = {
      name: "Organisation1",
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
      failOnStatusCode: false,
    }).then((response) => {
      expect(response.status).to.eq(422);
    });
  });
  it("should return an error if unauthorized user tries to create an organization", () => {
    const newOrganisation = {
      name: "Organisation1",
      description: "Organisation1 Desc",
      email: `org${Date.now()}@email.com`,
      type: "OrganisationType",
      location: "OrganisationLocation",
      country: "OrganisationCountry",
    };
    cy.request({
      method: "POST",
      url: `${baseUrl}/organisations`,
      body: newOrganisation,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer`,
      },
      failOnStatusCode: false,
    }).then((response) => {
      expect(response.status).to.eq(401);
    });
  });
  it("should get a new role within an organization successfully", () => {
    const newRole = {
      name: `New Role 9`,
      description: "New Role Description",
    };
    cy.request({
      method: "GET",
      url: `${baseUrl}/organisations/${organizationId}/roles`,
      body: newRole,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      expect(response.status).to.eq(200);
      expect(response.body).to.have.property("status", "success");
    });
  });

  it("should return an error if unauthorized user tries to create a role", () => {
    const newRole = {
      name: "Unauthorized Role",
      description: "Unauthorized Role Description",
    };
    cy.request({
      method: "GET",
      url: `${baseUrl}/organisations/${organizationId}/roles`,
      body: newRole,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer invalidtoken`,
      },
      failOnStatusCode: false,
    }).then((response) => {
      expect(response.status).to.eq(401);
      //expect(response.body).to.have.property("Unauthorized");
    });
  });
});
