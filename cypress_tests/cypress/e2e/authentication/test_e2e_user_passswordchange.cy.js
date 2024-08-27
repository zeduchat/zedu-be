describe('User change password',()=>{
    const baseUrl = Cypress.env("baseUrl");

    let authToken;
  
    const now = Date.now()
    const email = `my-test-user${now}@email.com`;
    const username = 'specialUsername';
    const password = 'password';
    const first_name = 'FirstName';
    const last_name = 'LastName'

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

it('should allow user change password',()=>{
    cy.request({
        method: "PUT",
        url: `${baseUrl}/auth/change-password`,
        failOnStatusCode: false,
        headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${authToken}`,
        },
        body:{
            "old_password": password,
            "new_password": `${password}12`
        }
    }).then(resp=>{
          expect(resp.status).to.eq(200);

    })
  
})  
})