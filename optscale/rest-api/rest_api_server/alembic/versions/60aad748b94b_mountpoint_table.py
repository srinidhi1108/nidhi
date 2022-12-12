""""mountpoint_table"

Revision ID: 60aad748b94b
Revises: 3e94569c2725
Create Date: 2017-07-07 11:06:36.801773

"""
import uuid
from alembic import op
import sqlalchemy as sa
from sqlalchemy.orm import Session
from sqlalchemy.sql import table, column
from rest_api_server.utils import Config


# revision identifiers, used by Alembic.
revision = '60aad748b94b'
down_revision = '3e94569c2725'
branch_labels = None
depends_on = None


def upgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    mountpoint_table = op.create_table('mountpoint',
    sa.Column('deleted_at', sa.Integer(), nullable=False),
    sa.Column('id', sa.String(length=36), nullable=False),
    sa.Column('mountpoint', sa.TEXT(), nullable=True),
    sa.Column('size', sa.BigInteger(), nullable=False),
    sa.PrimaryKeyConstraint('id')
    )
    mountpoints = Config().client.storages()
    mountpoint_records = []
    for mountpoint in mountpoints:
        mountpoint_record = {
            'deleted_at': 0,
            'id': str(uuid.uuid4()),
            'mountpoint': mountpoint,
            'size': 0,
        }
        mountpoint_records.append(mountpoint_record)
    op.bulk_insert(mountpoint_table, mountpoint_records)
    op.add_column('customer', sa.Column('mountpoint_id', sa.String(length=36), nullable=True))
    customer_table = table('customer', column('mountpoint_id',sa.String(36)))
    bind = op.get_bind()
    session = Session(bind=bind)
    upd_stmt = sa.update(customer_table).values(mountpoint_id=mountpoint_records[0]['id'])
    try:
        session.execute(upd_stmt)
        session.commit()
    finally:
        session.close()
    op.alter_column('customer', 'mountpoint_id', existing_type=sa.String(length=36), nullable=False)
    op.create_foreign_key('customer_ibfk_2', 'customer', 'mountpoint', ['mountpoint_id'], ['id'])
    # ### end Alembic commands ###


def downgrade():
    # ### commands auto generated by Alembic - please adjust! ###
    op.drop_constraint('customer_ibfk_2', 'customer', type_='foreignkey')
    op.drop_column('customer', 'mountpoint_id')
    op.drop_table('mountpoint')
    # ### end Alembic commands ###
