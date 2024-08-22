import { faker } from '@faker-js/faker';

describe('API Test - Successful User Registration', () => {
  const baseUrl = Cypress.env('baseURL');

  function generateUserData(overrides = {}) {
    return {
      username: faker.internet.userName(),
      email: faker.internet.email(),
      password: faker.internet.password(12, false, /[A-Za-z0-9!@#$%^&*()]/),
      first_name: faker.person.firstName(),
      last_name: faker.person.lastName(),
      ...overrides,
    };
  }

  beforeEach(() => {
    cy.intercept('POST', `${baseUrl}/auth/register`, (req) => {
      if (req.body.email && req.body.password && req.body.username) {
        req.reply({
          statusCode: 201,
          body: {
            status: 'success',
            message: 'user created successfully',
            status_code: 201,
          },
        });
      }
    }).as('registerUser');
  });

  it('should register a new user successfully', () => {
    const userData = generateUserData();

    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/register`,
      headers: {
        'Content-Type': 'application/json',
        accept: 'application/json',
      },
      body: userData,
    }).then((response) => {
      expect(response.status).to.eq(201);
      expect(response.body.status).to.eq('success');
      expect(response.body.message).to.eq('user created successfully');
      expect(response.body.status_code).to.eq(201);
    });
  });

  // For Bad Request with invalid input

  beforeEach(() => {
    cy.intercept('POST', `${baseUrl}/auth/register`, (req) => {
      // Check for invalid email format
      if (req.body.email && typeof req.body.email !== 'string') {
        req.reply({
          statusCode: 400,
          body: {
            status: 'Bad Request',
            status_code: 400,
            message: 'Invalid email format',
          },
        });
      }
    }).as('badRequestRegister');
  });

  it('should return 400 Bad Request for invalid email format', () => {
    const invalidUserData = generateUserData();
    invalidUserData.email = '3e'; 

    cy.request({
      method: 'POST',
      url: `${baseUrl}/auth/register`,
      headers: {
        'Content-Type': 'application/json',
        accept: 'application/json',
      },
      body: invalidUserData,
      failOnStatusCode: false,
    }).then((response) => {
      expect(response.status).to.eq(400);
      expect(response.body.status).to.eq('error');
      expect(response.body.status_code).to.eq(400);
      expect(response.body.message).to.eq('email address is invalid');
    });
  });

  // Unprocessed Entity with missing Field
   beforeEach(() => {
     cy.intercept('POST', `${baseUrl}/auth/register`, (req) => {
       if (!req.body.password) {
         req.reply({
           statusCode: 422,
           body: {
             status: 'error',
             status_code: 422,
             message: 'Validation failed',
             errors: [{ field: 'password' }],
           },
         });
       }
     }).as('unprocessedEntityRegister');
   });

   it('should return 422 Unprocessed Entity when password is missing', () => {
     const incompleteUserData = generateUserData();
     delete incompleteUserData.password; // Remove the password field

     cy.request({
       method: 'POST',
       url: `${baseUrl}/auth/register`,
       headers: {
         'Content-Type': 'application/json',
         accept: 'application/json',
       },
       body: incompleteUserData,
       failOnStatusCode: false,
     }).then((response) => {
       expect(response.status).to.eq(422);
       expect(response.body.status).to.eq('error');
       expect(response.body.status_code).to.eq(422);
       expect(response.body.message).to.eq('Validation failed');
       expect(response.body.errors).to.be.an('undefined');
     });
   });
});

