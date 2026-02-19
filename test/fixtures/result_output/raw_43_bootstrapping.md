---
provider: perplexity
mode: reason
mode_adjusted: false
truncated: false
sources_count: 7
cached: false
---

The search results provide guidance on async SQLAlchemy and Alembic setup, but lack comprehensive information on pytest-asyncio test fixtures and automatic rollback patterns. I'll synthesize what's available and note the gaps.

## Installation and Basic Setup

Start with the necessary dependencies:

```bash
pip install fastapi sqlalchemy[asyncio] asyncpg alembic uvicorn pytest pytest-asyncio
```

The `asyncpg` driver provides async support for PostgreSQL, which is preferred over `psycopg2` for async applications.

## Database Configuration

Create `src/db.py` with async engine and session management:

```python
from typing import AsyncGenerator
from sqlalchemy.ext.asyncio import AsyncEngine, AsyncSession, create_async_engine, async_sessionmaker
from sqlalchemy.pool import AsyncAdaptedQueuePool

# Load from environment variables
DATABASE_URL = "postgresql+asyncpg://postgres:password@localhost:5432/dbname"

engine: AsyncEngine = create_async_engine(
    DATABASE_URL,
    echo=True,
    poolclass=AsyncAdaptedQueuePool,
    pool_size=10,
    max_overflow=20,
    pool_pre_ping=True,
    pool_recycle=3600,
)

async_session_maker = async_sessionmaker(
    engine,
    class_=AsyncSession,
    expire_on_commit=False,
    autoflush=False,
)

async def get_async_db_session() -> AsyncGenerator[AsyncSession, None]:
    async with async_session_maker() as session:
        yield session

async def close_db() -> None:
    await engine.dispose()
```

Key parameters explained:
- **pool_size**: Maximum active connections (default: 5)
- **max_overflow**: Additional connections allowed beyond pool_size
- **pool_pre_ping**: Verifies connection health before reusing
- **pool_recycle**: Recycles connections after specified seconds (prevents idle connection closure)
- **expire_on_commit**: Keeps ORM objects alive after commit for access to lazy-loaded attributes

## Dependency Injection in FastAPI Routes

Use FastAPI's `Depends` to inject sessions:

```python
from fastapi import FastAPI, Depends
from sqlalchemy.ext.asyncio import AsyncSession

app = FastAPI()

@app.get("/users")
async def get_users(session: AsyncSession = Depends(get_async_db_session)):
    result = await session.execute(select(User))
    users = result.scalars().all()
    return users

@app.post("/users")
async def create_user(user: UserCreate, session: AsyncSession = Depends(get_async_db_session)):
    db_user = User(**user.dict())
    session.add(db_user)
    await session.commit()
    await session.refresh(db_user)
    return db_user
```

## Alembic Migrations Setup

Initialize Alembic:

```bash
alembic init alembic
```

Update `alembic/env.py` for async support:

```python
import asyncio
from sqlalchemy.ext.asyncio import async_engine_from_config
from sqlmodel import SQLModel
from app.models import Base  # Import your models

async def run_migrations_online() -> None:
    configuration = config.get_section(config.config_ini_section)
    configuration["sqlalchemy.url"] = os.getenv("DATABASE_URL")
    
    connectable = async_engine_from_config(
        configuration,
        prefix="sqlalchemy.",
        poolclass=AsyncAdaptedQueuePool,
    )

    async with connectable.begin() as connection:
        await connection.run_sync(do_run_migrations)

def do_run_migrations(connection):
    context.configure(connection=connection, target_metadata=Base.metadata)
    with context.begin_transaction():
        context.run_migrations()

asyncio.run(run_migrations_online())
```

## Test Database with pytest-asyncio

The search results do not contain detailed pytest-asyncio fixture patterns. However, the standard approach involves:

```python
# conftest.py
import pytest
import pytest_asyncio
from sqlalchemy.ext.asyncio import create_async_engine, AsyncSession, async_sessionmaker

TEST_DATABASE_URL = "postgresql+asyncpg://postgres:password@localhost:5432/test_db"

@pytest_asyncio.fixture
async def test_engine():
    """Create a test database engine."""
    engine = create_async_engine(TEST_DATABASE_URL, echo=False)
    
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    
    yield engine
    
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.drop_all)
    
    await engine.dispose()

@pytest_asyncio.fixture
async def test_session(test_engine):
    """Provide a test session with automatic rollback."""
    async_session_maker = async_sessionmaker(test_engine, class_=AsyncSession, expire_on_commit=False)
    
    async with async_session_maker() as session:
        async with session.begin_nested():
            yield session
            await session.rollback()

@pytest_asyncio.fixture
def override_get_db(test_session):
    """Override dependency for tests."""
    async def _get_db():
        yield test_session
    return _get_db
```

Then in your tests:

```python
@pytest.mark.asyncio
async def test_create_user(override_get_db):
    app.dependency_overrides[get_async_db_session] = override_get_db
    
    client = TestClient(app)
    response = client.post("/users", json={"name": "John"})
    assert response.status_code == 201
    
    app.dependency_overrides.clear()
```

## Configuration Best Practices

Store credentials in environment variables rather than hardcoding them. Use a settings file:

```python
# settings.py
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    POSTGRES_USER: str
    POSTGRES_PASSWORD: str
    POSTGRES_HOST: str
    POSTGRES_PORT: int
    POSTGRES_DB: str

settings = Settings()
```

**Important gap**: The provided search results do not include comprehensive patterns for pytest-asyncio fixtures with automatic transaction rollback. The fixture pattern shown above represents best practices not covered in your search results and should be validated against your specific testing requirements.