import { faker } from '@faker-js/faker';

describe('Create organisations API Tests', () => {
  const baseUrl = Cypress.env('baseURL');
  let authToken;
  let email;
  let password;

  before(() => {
    email = faker.internet.email();
    password = faker.internet.password(12) + 'A1!';
    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/register`,
      body: {
        username: faker.internet.userName(),
        email: email,
        password: password,
        first_name: faker.person.firstName(),
        last_name: faker.person.lastName(),
      },
      headers: {
        'Content-Type': 'application/json',
      },
      failOnStatusCode: false,
    }).then((response) => {
      expect(response.status).to.be.oneOf([200, 201]);
      expect(response.body).to.have.property('status', 'success');
    });
    cy.wait(5000);
    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/login`,
      body: {
        email: email,
        password: password,
      },
      headers: {
        'Content-Type': 'application/json',
      },
      failOnStatusCode: false,
    }).then((response) => {
      cy.log(JSON.stringify(response.body));
      if (response.status === 200 && response.body.status === 'success') {
        authToken = response.body.data.access_token;
      } else {
        cy.log(
          `Login failed. Status: ${response.status}, Body: ${JSON.stringify(
            response.body
          )}`
        );
        authToken = 'dummy_token';
      }
    });
  });

  it('should attempt to logout', () => {
    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/logout`,
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${authToken}`,
      },
      failOnStatusCode: false,
    }).then((response) => {
      cy.log(`Logout response: ${JSON.stringify(response.body)}`);
    });
  });

  it('should attempt to access protected route', () => {
    cy.request({
      method: 'GET',
      url: `${baseUrl}/some-protected-route`, 
      headers: {
        Authorization: `Bearer ${authToken}`,
      },
      failOnStatusCode: false,
    }).then((response) => {
      cy.log(`Protected route response: ${JSON.stringify(response.body)}`);
    });
  });

  afterEach(() => {
    cy.wait(1000);
  });
});
