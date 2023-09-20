#!/usr/bin/env node
const express = require('express');
const url = require('url');
const argv = require('process').argv;
const Buffer = require('buffer').Buffer;

// Validate command-line arguments
if (argv.length < 3) {
  console.error('Usage: run-graphiql [GraphQL endpoint URL]');
  process.exit(1);
}

let originalGraphqlEndpoint = argv[2];
const parsedUrl = new url.URL(originalGraphqlEndpoint);
const username = parsedUrl.username;
const password = parsedUrl.password;

// Remove user credentials from the URL
parsedUrl.username = "";
parsedUrl.password = "";
const strippedGraphqlEndpoint = parsedUrl.toString();

// Generate Basic Authentication header if credentials are present
const basicAuthHeader = username ? `Basic ${Buffer.from(`${username}:${password}`).toString('base64')}` : '';
const authHeaderLine = basicAuthHeader ? `'Authorization': '${basicAuthHeader}',` : '';

const port = 4000;
const app = express();

app.get('/graphiql', (req, res) => {
  const graphiqlHtml = `
    <!DOCTYPE html>
    <html>
      <head>
        <title>GraphiQL</title>
        <link href="https://unpkg.com/graphiql/graphiql.min.css" rel="stylesheet" />
      </head>
      <body>
        <div id="graphiql" style="height: 100vh;"></div>
        <script crossorigin src="https://unpkg.com/react/umd/react.production.min.js"></script>
        <script crossorigin src="https://unpkg.com/react-dom/umd/react-dom.production.min.js"></script>
        <script crossorigin src="https://unpkg.com/graphiql/graphiql.min.js"></script>
        <script>
          function graphQLFetcher(graphQLParams) {
            return fetch('${strippedGraphqlEndpoint}', {
              method: 'post',
              headers: {
                'Accept': 'application/json',
                'Content-Type': 'application/json',
                ${authHeaderLine}
              },
              body: JSON.stringify(graphQLParams),
              credentials: 'omit',
            })
            .then(response => {
              if (response.ok) {
                return response.json();
              }
              return response.text().then(bodyText => {
                throw new Error(\`HTTP error \${response.status} - \${bodyText}\`);
              });
            });
          }

          ReactDOM.render(
            React.createElement(GraphiQL, { fetcher: graphQLFetcher }),
            document.getElementById('graphiql'),
          );
        </script>
      </body>
    </html>
  `;

  res.send(graphiqlHtml);
});

app.listen(port, () => {
  console.log(`GraphiQL is running at http://localhost:${port}/graphiql\nendpoint: ${strippedGraphqlEndpoint}\nusername: ${username}\nauth header: ${authHeaderLine}`);
});
