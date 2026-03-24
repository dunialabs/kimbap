#!/usr/bin/env node

const { execSync } = require('child_process')

function runCommandWithOutput(command) {
  try {
    const output = execSync(command, {
      encoding: 'utf8',
      stdio: 'pipe'
    })
    if (output) {
      process.stdout.write(output)
    }
    return { ok: true, output: output || '' }
  } catch (error) {
    const stdout = error.stdout ? error.stdout.toString() : ''
    const stderr = error.stderr ? error.stderr.toString() : ''
    const output = `${stdout}${stderr}`
    if (output) {
      process.stderr.write(output)
    }
    return { ok: false, error, output }
  }
}

async function setupDatabase() {
  console.log('🔧 Setting up database...')

  try {
    // Check if PostgreSQL container is running, if not start it
    console.log('🐳 Checking PostgreSQL container status...')
    try {
      execSync('docker ps --filter "name=kimbap-postgres" --filter "status=running" | grep kimbap-postgres', {
        stdio: 'pipe'
      })
      console.log('✅ PostgreSQL container is already running')
    } catch (error) {
      console.log('🚀 Starting PostgreSQL container...')
      try {
        execSync('docker compose up -d postgres', {
          stdio: 'inherit'
        })
        console.log('✅ PostgreSQL container started')
        // Wait a bit for container to initialize
        await new Promise((resolve) => setTimeout(resolve, 2000))
      } catch (startError) {
        console.error('❌ Failed to start PostgreSQL container:', startError.message)
        throw new Error('Failed to start PostgreSQL container. Make sure Docker is running.')
      }
    }

    // Wait for PostgreSQL to be ready
    console.log('⏳ Waiting for PostgreSQL to be ready...')
    let retries = 0
    const maxRetries = 30

    while (retries < maxRetries) {
      try {
        execSync('docker exec kimbap-postgres pg_isready -U kimbap', {
          stdio: 'pipe'
        })
        break
      } catch (error) {
        retries++
        if (retries >= maxRetries) {
          throw new Error('PostgreSQL failed to start after 30 attempts')
        }
        await new Promise((resolve) => setTimeout(resolve, 1000))
      }
    }

    console.log('✅ PostgreSQL is ready')

    // Test database connection
    console.log('🔍 Testing database connection...')
    try {
      execSync(
        'docker exec kimbap-postgres psql -U kimbap -d kimbap_db -c "SELECT 1;"',
        { stdio: 'pipe' }
      )
      console.log('✅ Database connection successful')
    } catch (error) {
      console.log(
        '❌ Database connection failed, attempting to create database...'
      )
      try {
        execSync(
          'docker exec kimbap-postgres psql -U kimbap -c "CREATE DATABASE kimbap_db;"',
          { stdio: 'pipe' }
        )
        console.log('✅ Database created successfully')
      } catch (createError) {
        console.log('ℹ️  Database might already exist, continuing...')
      }
    }

    // Apply Prisma migrations (avoid mixing db push + migrate deploy)
    console.log('📦 Applying Prisma migrations...')
    const firstDeploy = runCommandWithOutput(
      'npx prisma migrate deploy --schema=./prisma/schema.prisma'
    )
    if (!firstDeploy.ok) {
      const errorOutput = firstDeploy.output

      const hasKnownFailedMigration =
        errorOutput.includes('P3009') &&
        errorOutput.includes('20260213190000_add_token_metadata')

      let tokenMetadataTableExists = false
      if (hasKnownFailedMigration) {
        try {
          const tableExistsResult = execSync(
            'docker exec kimbap-postgres psql -U kimbap -d kimbap_db -tAc "SELECT to_regclass(\'public.token_metadata\') IS NOT NULL;"',
            {
              encoding: 'utf8',
              stdio: 'pipe'
            }
          )
          tokenMetadataTableExists = tableExistsResult.trim() === 't'
        } catch (_checkError) {
          tokenMetadataTableExists = false
        }
      }

      const isKnownTokenMetadataConflict =
        hasKnownFailedMigration && tokenMetadataTableExists

      if (!isKnownTokenMetadataConflict) {
        throw firstDeploy.error
      }

      console.log(
        '⚠️ Detected pre-existing token_metadata table, resolving migration state...'
      )
      execSync(
        'npx prisma migrate resolve --applied "20260213190000_add_token_metadata" --schema=./prisma/schema.prisma',
        { stdio: 'inherit' }
      )

      const secondDeploy = runCommandWithOutput(
        'npx prisma migrate deploy --schema=./prisma/schema.prisma'
      )
      if (!secondDeploy.ok) {
        throw secondDeploy.error
      }
    }
    console.log('✅ Database schema is up to date')

    console.log('🎉 Database setup completed successfully!')
  } catch (error) {
    console.error('❌ Database setup failed:', error.message)
    process.exit(1)
  }
}

if (require.main === module) {
  setupDatabase()
}

module.exports = { setupDatabase }
