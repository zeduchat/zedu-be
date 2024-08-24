describe("Create Channel", () => {
    const baseUrl = Cypress.env("baseUrl");
    let authToken;
    let organisation_id;
    const now = Date.now();
    const email = `user${now}@example.com`;
    const username = `user${now}`;
    const password = "password"; 
  
    before(() => {
      // Sign up user
      cy.request({
        method: "POST",
        url: `${baseUrl}/auth/register`,
        body: {
          username,
          email,
          password,
          first_name: "FirstName",
          last_name: "LastName",
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
          email,
          password,
        },
        headers: {
          "Content-Type": "application/json",
        },
      }).then((response) => {
        expect(response.status).to.eq(200);
        authToken = response.body.data.access_token;

        const newOrganisation = {
          name: `Organisation-${now}`,
          description: "Organisation Desc",
          email,
          type: "OrganisationType",
          location: "OrganisationLocation",
          country: "OrganisationCountry",
        };
    
        // Create an organization
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
          organisation_id = response.body?.data?.id
        });

      });

    });
  
    it("should create a new channel successfully", () => {

      const newChannel = {
        name: `Channel-${now}`,
        description: "This is a test channel",
        organisation_id, 
        username: username,
      };
  
      cy.request({
        method: "POST",
        url: `${baseUrl}/channels`,
        body: newChannel,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}`,
        },
      }).then((response) => {
        expect(response.status).to.eq(201);
        expect(response.body.data.name).to.eq(newChannel.name);
        expect(response.body.data.description).to.eq(newChannel.description);
      });
    });
  
    it("should fail if required fields are missing", () => {
      cy.request({
        method: "POST",
        url: `${baseUrl}/channels`,
        body: {}, // Missing required fields
        failOnStatusCode: false,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}`,
        },
      }).then((response) => {
        cy.log(`Status code: ${response.status}`);
        cy.log(`Response body: ${JSON.stringify(response.body)}`);
        expect(response.status).to.be.oneOf([400, 422]);
      });
    });
  
    it("should fail if the user is not authenticated", () => {
      cy.request({
        method: "POST",
        url: `${baseUrl}/channels`,
        body: {
          name: `Channel-${now}`,
          description: "This is a test channel",
          organisation_id: "3fa85f64-5717-4562-b3fc-2c963f66afa6",
          username: username,
        },
        failOnStatusCode: false,
        headers: {
          "Content-Type": "application/json",
        },
      }).then((response) => {
        cy.log(`Status code: ${response.status}`);
        cy.log(`Response body: ${JSON.stringify(response.body)}`);
        expect(response.status).to.eq(401);
      });
    });
  
    it("should fail if the user does not have permission", () => {
      
      cy.request({
        method: "POST",
        url: `${baseUrl}/channels`,
        body: {
          name: `Channel-${now}`,
          description: "This is a test channel",
          organisation_id: "3fa85f64-5717-4562-b3fc-2c963f66afa6",
          username: username,
        },
        failOnStatusCode: false,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${authToken}`, 
        },
      }).then((response) => {
        cy.log(`Status code: ${response.status}`);
        cy.log(`Response body: ${JSON.stringify(response.body)}`);
        expect(response.status).to.be.oneOf([400, 403]);
      });
    });
  });
  