""""removed_parked_from_process"

Revision ID: 70d6376c23a0
Revises: 4b2cd5e698dd
Create Date: 2018-04-11 10:25:29.527362

"""
from alembic import op
from sqlalchemy.orm import Session

# revision identifiers, used by Alembic.
revision = '70d6376c23a0'
down_revision = '4b2cd5e698dd'
branch_labels = None
depends_on = None


def upgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    bind = op.get_bind()
    session = Session(bind=bind)
    try:
        session.execute("ALTER TABLE device_state MODIFY COLUMN process "
                        "ENUM('replicating', 'canceling', 'idle');")
    finally:
        session.close()
    # ### end Alembic commands ###


def downgrade():
    # ### commands auto generated by Alembic - please adjust! ###

    bind = op.get_bind()
    session = Session(bind=bind)

    try:
        session.execute("ALTER TABLE device_state MODIFY COLUMN process "
                        "ENUM('replicating', 'canceling', 'idle', 'parked');")
    finally:
        session.close()
    # ### end Alembic commands ###
