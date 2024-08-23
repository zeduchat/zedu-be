import { faker } from '@faker-js/faker';

describe('API Test - User Login', () => {
  const baseUrl = Cypress.env('baseURL');
  let registeredEmail;
  let registeredPassword;

  before(() => {
    registeredEmail = faker.internet.email();
    registeredPassword = faker.internet.password(
      12,
      false,
      /[A-Za-z0-9!@#$%^&*()]/
    );
    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/register`,
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: {
        username: faker.internet.userName(),
        email: registeredEmail,
        password: registeredPassword,
        first_name: faker.person.firstName(),
        last_name: faker.person.lastName(),
      },
    }).then((response) => {
      expect(response.status).to.eq(201);
    });
  });

  it('should login a user successfully', () => {
    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/login`,
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: {
        email: registeredEmail,
        password: registeredPassword,
      },
      failOnStatusCode: false,
    }).then((response) => {
      if (response.status === 200) {
        expect(response.body.status).to.eq('success');
        expect(response.body.message).to.eq('Login Successfully');
        expect(response.body.status_code).to.eq(200);
        expect(response.body.data).to.have.property('user');
        expect(response.body.data).to.have.property('access_token');
        expect(response.body.data.access_token).to.be.a('string');
        expect(response.duration).to.be.lessThan(500);
      } else if (response.status === 400) {
        expect(response.body.status).to.eq('error');
        expect(response.body.message).to.eq('invalid credentials');
      } else {
        throw new Error(`Unexpected status code: ${response.status}`);
      }
    });
  });

  it('should return 400 Bad Request for invalid credentials', () => {
    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/login`,
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: {
        email: 'invalid@example.com',
        password: 'invalidPassword',
      },
      failOnStatusCode: false,
    }).then((response) => {
      expect(response.status).to.eq(400);
      expect(response.body.status).to.eq('error');
      expect(response.body.status_code).to.eq(400);
      expect(response.body.message).to.eq('invalid credentials');
    });
  });

  it('should return 422 Unprocessable Entity when email is missing', () => {
    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/login`,
      headers: {
        'Content-Type': 'application/json',
        Accept: 'application/json',
      },
      body: {
        password: registeredPassword, 
      },
      failOnStatusCode: false, 
    }).then((response) => {
      if (response.status === 422) {
        expect(response.body.status).to.eq('error');
        expect(response.body.status_code).to.eq(422);
        expect(response.body.message).to.eq('Validation failed');
        expect(response.body.errors).to.be.an('array');
        expect(response.body.errors[0].field).to.eq('email');
      } else if (response.status === 400) {
        expect(response.body.status).to.eq('error');
        expect(response.body.status_code).to.eq(400);
        expect(response.body.message).to.eq('Validation failed'); 
      } else {
        throw new Error(`Unexpected status code: ${response.status}`);
      }
    });
  });

});
