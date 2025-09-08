# Contributing to Go Chat Server

Thank you for your interest in contributing to Go Chat Server! This document provides guidelines and information for contributors.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

## Getting Started

### Prerequisites

- Go 1.21 or later
- Git
- Make (optional but recommended)

### Development Setup

1. Fork the repository on GitHub
2. Clone your fork:
```bash
git clone https://github.com/your-username/go-chat.git
cd go-chat
```

3. Add the upstream remote:
```bash
git remote add upstream https://github.com/original-owner/go-chat.git
```

4. Install dependencies:
```bash
go mod download
```

5. Set up development environment:
```bash
make generate-cert    # Generate TLS certificates
make generate-secret  # Generate JWT secret
```

6. Run tests to verify setup:
```bash
go test ./...
```

## Development Workflow

### Branching Strategy

- `main` - Production-ready code
- Feature branches - `feat/feature-name`
- Bug fixes - `fix/bug-description`
- Documentation - `docs/update-description`

### Making Changes

1. Create a feature branch:
```bash
git checkout -b feat/your-feature-name
```

2. Make your changes following our coding standards
3. Write or update tests for your changes
4. Run the test suite:
```bash
go test ./...
```

5. Run linting and formatting:
```bash
go fmt ./...
go vet ./...
```

6. Commit your changes using conventional commits (see below)
7. Push to your fork and create a pull request

## Coding Standards

### Go Style Guidelines

- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Run `go vet` to catch common issues
- Use meaningful variable and function names
- Add comments for exported functions and types

### Project Structure

- `cmd/` - Application entry points
- `internal/` - Private application code
- `pkg/` - Public library code
- `docs/` - Generated documentation (do not edit manually)

### Testing

We follow Test-Driven Development (TDD):

1. Write failing tests first
2. Implement minimal code to make tests pass
3. Refactor while keeping tests green

#### Test Guidelines

- Use table-driven tests when appropriate
- Test both success and error cases
- Use descriptive test names: `TestFunctionName_Condition_ExpectedResult`
- Mock external dependencies
- Keep tests focused and independent

#### Example Test Structure

```go
func TestUserService_UpdateUser_Success(t *testing.T) {
    // Arrange
    db := setupTestDB(t)
    service := user.NewService(db)
    
    // Act
    err := service.UpdateUser(ctx, userID, updateReq)
    
    // Assert
    assert.NoError(t, err)
    // Additional assertions...
}
```

## Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `test` - Adding or updating tests
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `chore` - Maintenance tasks

### Examples

```
feat: add user management endpoints

Add PATCH /api/user for updating username/password
Add DELETE /api/user for account deletion
Includes comprehensive test coverage
```

```
fix: resolve rate limiting memory leak

Fix cleanup routine in IPRateLimiter to properly
remove unused limiters based on token availability
```

## API Design Guidelines

### RESTful Conventions

- Use HTTP methods correctly (GET, POST, PATCH, DELETE)
- Use plural nouns for resources (`/api/users`, `/api/channels`)
- Use nested routes for sub-resources (`/api/channels/:id/users`)
- Return appropriate HTTP status codes

### Request/Response Format

- Use JSON for request and response bodies
- Include proper validation with descriptive error messages
- Use consistent error response format:

```json
{
  "error": "validation_failed",
  "message": "Username is required",
  "details": {
    "field": "username",
    "code": "required"
  }
}
```

### Authentication

- All protected endpoints require JWT authentication
- Include rate limiting considerations
- Use appropriate HTTP status codes (401, 403)

## Database Guidelines

### Migrations

- Use GORM's auto-migration feature
- Test migrations with sample data
- Consider backward compatibility

### Models

- Define models in `pkg/chat/models.go`
- Use appropriate GORM tags
- Include validation tags where needed
- Use nanoid for IDs (8 chars for users, 6 for channels)

## Documentation

### API Documentation

- Use Swagger/OpenAPI annotations
- Include examples in request/response schemas
- Document all error responses
- Regenerate docs after API changes:

```bash
swag init -g cmd/server/main.go -o docs/
```

### Code Documentation

- Document all exported functions and types
- Use clear, concise comments
- Include usage examples for complex functions

## Security Considerations

### Authentication & Authorization

- Always validate JWT tokens on protected endpoints
- Implement proper role-based access control
- Use bcrypt for password hashing

### Rate Limiting

- Apply appropriate rate limits based on endpoint sensitivity
- Consider different limits for authenticated vs anonymous users
- Test rate limiting functionality

### Input Validation

- Validate all user inputs
- Sanitize data before database operations
- Use parameterized queries to prevent SQL injection

## Performance Guidelines

### Database

- Use appropriate indexes
- Limit query results with pagination
- Avoid N+1 query problems

### HTTP

- Implement compression for large responses
- Use appropriate caching headers
- Consider connection pooling for database

## Pull Request Process

1. Ensure all tests pass
2. Update documentation if needed
3. Add tests for new functionality
4. Follow the commit message format
5. Provide a clear PR description explaining:
   - What changes were made
   - Why they were necessary
   - How to test the changes

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass locally
- [ ] Added tests for new functionality
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes without version bump
```

## Getting Help

- Check existing issues and discussions
- Create a new issue for bugs or feature requests
- Join our community discussions
- Ask questions in pull request comments

## Release Process

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create release PR
4. Tag release after merge
5. Generate release notes

Thank you for contributing to Go Chat Server! 🎉