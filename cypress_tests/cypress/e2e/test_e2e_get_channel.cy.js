describe("To retreive channel", () => {
  const baseUrl = Cypress.env("baseURL");
  let authToken;
  let channelId;
  let organisation_id;
  const now = Date.now();
  const email = `user${now}@example.com`;
  const username = `user${now}`;
  const password = "password";
  const newOrganisation = {
    name: `Organisation-${now}`,
    description: "Organisation Desc",
    email,
    type: "OrganisationType",
    location: "OrganisationLocation",
    country: "OrganisationCountry",
  };
  const newChannel = {
    name: `Channel-${now}`,
    description: "This is a test channel",
    username: username,
  };
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
        organisation_id = response.body?.data?.id;

        // create a channel
        newChannel.organisation_id = organisation_id;

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
          channelId = response.body.data.channels_id;
        });
      });
    });
  });

  it("should retrieve the created channel successfully", () => {
    cy.request({
      method: "GET",
      url: `${baseUrl}/channels/${channelId}`,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
    }).then((response) => {
      expect(response.status).to.eq(200);
      expect(response.body.data).to.have.property("channels_id", channelId);
      expect(response.body.data).to.have.property("name", newChannel.name);
      expect(response.body.data).to.have.property(
        "description",
        newChannel.description
      );
      expect(response.body.data).to.have.property(
        "organisation_id",
        organisation_id
      );
    });
  });

  it("should fail to retrieve a channel if not authenticated", () => {
    cy.request({
      method: "GET",
      url: `${baseUrl}/channels/${channelId}`,
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

  it("should fail to retrieve a channel if the channel does not exist", () => {
    cy.request({
      method: "GET",
      url: `${baseUrl}/channels/invalid-channel-id`,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
      },
      failOnStatusCode: false,
    }).then((response) => {
      cy.log(`Status code: ${response.status}`);
      cy.log(`Response body: ${JSON.stringify(response.body)}`);
      expect(response.status).to.eq(400);
    });
  });
});
