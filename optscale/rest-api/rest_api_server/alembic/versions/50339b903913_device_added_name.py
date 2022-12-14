""""device_added_name"

Revision ID: 50339b903913
Revises: 683f50a0f0f7
Create Date: 2017-05-19 14:54:57.136759

"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '50339b903913'
down_revision = '683f50a0f0f7'
branch_labels = None
depends_on = None


def upgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    op.add_column('device', sa.Column('name', sa.String(length=256), nullable=True))
    # ### end Alembic commands ###


def downgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    op.drop_column('device', 'name')
    # ### end Alembic commands ###
