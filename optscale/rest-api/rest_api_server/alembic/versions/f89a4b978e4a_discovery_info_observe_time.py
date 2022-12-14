""""discovery_info_observe_time"

Revision ID: f89a4b978e4a
Revises: 9c2012e13f95
Create Date: 2021-11-01 06:55:19.552810

"""
from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision = 'f89a4b978e4a'
down_revision = '9c2012e13f95'
branch_labels = None
depends_on = None


def upgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    op.add_column('discovery_info', sa.Column(
        'observe_time', sa.Integer(), nullable=False))
    # ### end Alembic commands ###


def downgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    op.drop_column('discovery_info', 'observe_time')
    # ### end Alembic commands ###
