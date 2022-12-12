""""ssh_keys"

Revision ID: a122b4004241
Revises: 3b13e59e69b1
Create Date: 2021-10-28 10:50:04.650260

"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import mysql

# revision identifiers, used by Alembic.
revision = 'a122b4004241'
down_revision = '3b13e59e69b1'
branch_labels = None
depends_on = None


def upgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    op.create_table(
        'ssh_key',
        sa.Column('deleted_at', sa.Integer(), nullable=False),
        sa.Column('id', sa.String(36), nullable=False),
        sa.Column('created_at', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(256), nullable=False),
        sa.Column('employee_id', sa.String(36), nullable=False),
        sa.Column('default', sa.Boolean(), nullable=False),
        sa.Column('fingerprint', sa.String(256), nullable=False),
        sa.Column('key', sa.TEXT(), nullable=False),
        sa.ForeignKeyConstraint(['employee_id'], ['employee.id'], ),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('employee_id', 'fingerprint', 'deleted_at',
                            name='uc_employee_id_fingerprint_deleted_at')
    )
    op.add_column(
        'shareable_booking', sa.Column('ssh_key', sa.TEXT(), nullable=True))
    # ### end Alembic commands ###


def downgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    op.drop_column('shareable_booking', 'ssh_key')
    op.drop_table('ssh_key')
    # ### end Alembic commands ###
