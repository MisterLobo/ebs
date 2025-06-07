[![Deploy Server to Amazon ECS](https://github.com/MisterLobo/ebs/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/MisterLobo/ebs/actions/workflows/ci.yml)

# EventBookingSystem

<a alt="Nx logo" href="https://nx.dev" target="_blank" rel="noreferrer"><img src="https://raw.githubusercontent.com/nrwl/nx/master/images/nx-logo.png" width="45"></a>

✨ Your new, shiny [Nx workspace](https://nx.dev) is almost ready ✨.

[Learn more about this workspace setup and its capabilities](https://nx.dev/nx-api/next?utm_source=nx_project&amp;utm_medium=readme&amp;utm_campaign=nx_projects) or run `bunx nx graph` to visually explore what was created. Now, let's get you up to speed!

# ABOUT
Live Demo: https://app.silver-elven.cc

Silver Elven is an event booking system that aims to provide a platform for hosting events, buying and selling tickets with a simple and easy-to-use user interface. This platform is focused on events that are open and do not require reservations that depend on seating arrangements. In other words, this is intended for a first-come first-served or General Admission events only.

## Project Details
This is a monorepo using Nx for running, testing and managing projects
- `ebs-app`: frontend web application
- `api`: web server API
- `mobile`: Android mobile application using Flutter

### Tech Stack
- Web Application: NextJS, TailwindCSS, Shadcn, Radix
- Web Server: Go using Gin framework
- Mobile App: Flutter (tested with Android only)
- Services integrated:
  - Stripe for Payments
  - AWS (ECS, EventBridge Scheduler, SQS, SNS, RDS, S3, ECR, EC2, VPC, SES)
  - Vercel for hosting the web app
  - Redis for caching
  - Firebase for user and authentication
  - Cloudflare for anti-spam and bot protection

## FEATURES
- Host events
- Wait list
- Scheduled opening of registrations and start of admissions
- Buy and sell tickets
- Pay using Stripe
- QR code generation for tickets
- Includes mobile app for scanning and verifying QR code
- Download and share e-tickets
- Create multiple organizations

## TODO
- sales calculation and dashboard
- Trending events page
- Transfer tickets
- Multi tenancy, RBAC
- Swagger API docs
- UI/UX improvements

## LIMITATIONS
- Events hosted are only for General Admission types.
- No seating arrangement planner

## NOTES
- This project currently undergoes heavy testing to ensure it becomes production-ready.
If you like this project and plan to use it for commercial purposes feel free to fork this repo and set it up for production.
Also a star would be much appreciated
- This is a personal project. I built this project so I can explore and learn more about other languages and frameworks specially on other platforms.
- The codebase needs some refactoring specially in the `api` since I am still learning about the best practices of writing Go code. There are parts that you might find a bit spaghetti-ish but I try to organize and break down the application into modules.
- Requests to the API must include `x-secret` and `origin` in the headers. This is a temporary protection layer for the request but does not guarantee data security against malicious attacks. Only use for testing purposes.
- Implement a more secure protection that adhere to web security standards to remove the need for such headers in a production environment.

# DEVELOPMENT
Fork this repo and clone it
## Requirements:
- Flutter - [docs](https://docs.flutter.dev/get-started/install)
- `bun` - [docs](https://bun.sh/docs/installation)
- `nx` CLI tool - [docs](https://nx.dev/getting-started/installation)
- `stripe-mock` - [repo](https://github.com/stripe/stripe-mock)
- Firebase emulator suite for testing - [docs](https://firebase.google.com/docs/emulator-suite/install_and_configure)
- PostgeSQL
- Redis
- Kafka - run `docker-compose up` using [docker-compose.yml](https://github.com/MisterLobo/ebs/docker-compose.yml) at the root of the repository
- Install [atlas cli](https://atlasgo.io/guides/orms/gorm/getting-started) for gorm
- AWS account to use its services (ECS, EC2, ECR, SQS, SNS, S3, EventBridgeScheduler, etc.)

## Setup
- for `ebs-app` install the dependencies with `bun install`
- cd into `apps/api` and run `go mod tidy && go mod download`
- cd into `apps/mobile` and run `flutter pub get`

## Database Migration
- Run the scripts located at `apps/api/scripts` to manage migrations
- If you use AWS RDS, you need to set up the ECS containers, RDS cluster and EC2 instance in the same VPC which is the recommended way and more secure than exposing the database publicly.
- The EC2 instance will serve as the jumpbox for connecting to the RDS cluster from your local machine via SSH tunneling
- Usage of RDS is not required. You can use any cloud db provider such as Neon or Supabase.

## Run tasks

To run the dev server for your app, use:

```sh
nx dev ebs-app
```

View app in browser:
```sh
http://localhost:3000
```

To run dev server for api, use:
```sh
nx serve api
```
Health check
```sh
curl http://localhost:9090
```
API available at `/api/v1`

To create a production bundle:

```sh
nx build ebs-app
```

```sh
nx build api
```

```sh
nx build mobile
```

To see all available targets to run for a project, run:

```sh
nx show project ebs-app
```

## Testing
#### Tests are currently being set up to spot bugs and ensure full code coverage
```sh
nx test ebs-app
```
```sh
nx test api
```
```sh
nx test mobile
```

## Deployment
- AWS services are used an integrated in the API. You need to have an account with root privileges
- You can delete `fly.toml` in the `apps/api/` folder if you do not want to deploy to fly.io

## Production
- This project is not production-ready
- You need to have a live Stripe account
- You need to configure the release build of the mobile app if you wish to deploy to Play Store
- Only Android is configured to be built
- Need more testing

These targets are either [inferred automatically](https://nx.dev/concepts/inferred-tasks?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects) or defined in the `project.json` or `package.json` files.

[More about running tasks in the docs &raquo;](https://nx.dev/features/run-tasks?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects)

## Add new projects

While you could add new projects to your workspace manually, you might want to leverage [Nx plugins](https://nx.dev/concepts/nx-plugins?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects) and their [code generation](https://nx.dev/features/generate-code?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects) feature.

Use the plugin's generator to create new projects.

You can use `nx list` to get a list of installed plugins. Then, run `nx list <plugin-name>` to learn about more specific capabilities of a particular plugin. Alternatively, [install Nx Console](https://nx.dev/getting-started/editor-setup?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects) to browse plugins and generators in your IDE.

[Learn more about Nx plugins &raquo;](https://nx.dev/concepts/nx-plugins?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects) | [Browse the plugin registry &raquo;](https://nx.dev/plugin-registry?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects)


[Learn more about Nx on CI](https://nx.dev/ci/intro/ci-with-nx#ready-get-started-with-your-provider?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects)

## Install Nx Console

Nx Console is an editor extension that enriches your developer experience. It lets you run tasks, generate code, and improves code autocompletion in your IDE. It is available for VSCode and IntelliJ.

[Install Nx Console &raquo;](https://nx.dev/getting-started/editor-setup?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects)

## Useful links

Learn more:

- [Learn more about this workspace setup](https://nx.dev/nx-api/next?utm_source=nx_project&amp;utm_medium=readme&amp;utm_campaign=nx_projects)
- [Learn about Nx on CI](https://nx.dev/ci/intro/ci-with-nx?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects)
- [Releasing Packages with Nx release](https://nx.dev/features/manage-releases?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects)
- [What are Nx plugins?](https://nx.dev/concepts/nx-plugins?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects)

And join the Nx community:
- [Discord](https://go.nx.dev/community)
- [Follow us on X](https://twitter.com/nxdevtools) or [LinkedIn](https://www.linkedin.com/company/nrwl)
- [Our Youtube channel](https://www.youtube.com/@nxdevtools)
- [Our blog](https://nx.dev/blog?utm_source=nx_project&utm_medium=readme&utm_campaign=nx_projects)

## LICENSE
MIT