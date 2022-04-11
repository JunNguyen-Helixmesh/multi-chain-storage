create table wallet (
    id            bigint       not null auto_increment,
    address       varchar(100) not null,
    type          int          not null,  #--0:pay admin fee, 1:dao, 2:user,1000:file coin
    create_at     bigint       not null,
    primary key pk_wallet(id)
);

create table miner (
    id            bigint       not null auto_increment,
    fid           varchar(100) not null,
    create_at     bigint       not null,
    primary key pk_miner(id)
);

create table network (
    id            bigint        not null auto_increment,
    name          varchar(100)  not null,
    rpc_url       varchar(1000) not null,
    description   text
    create_at     bigint        not null,
    update_at     bigint        not null,
    primary key pk_network(id),
    constraint un_network_name unique(name)
);


create table coin (
    id            bigint       not null auto_increment,
    name          varchar(100) not null,
    network_id    bigint       not null,
    description   text
    create_at     bigint       not null,
    update_at     bigint       not null,
    primary key pk_coin(id),
    constraint un_coin_name unique(name),
    constraint fk_coin_network_id foreign key (network_id) references network(id)
)

create table source_file (
    id            bigint        not null auto_increment,
    file_type     int           not null,  #--0:normal file, 1:mint file
    status        varchar(100)  not null,
    uuid          varchar(100)  not null,
    payload_cid   varchar(100)  not null,
    resource_uri  varchar(1000) not null,
    ipfs_url      varchar(1000) not null,
    file_size     bigint        not null,
    dataset       varchar(100),
    pin_status    varchar(100)  not null,
    create_at     bigint        not null,
    update_at     bigint        not null,
    primary key pk_source_file(id),
    constraint un_source_file_resource_uri unique(resource_uri),
    constraint un_source_file_uuid_payload_cid unique(uuid,payload_cid),
    constraint fk_source_file_wallet_id foreign key (wallet_id) references wallet(id)
);

create table source_file_mint_info (
    id             bigint        not null auto_increment,
    source_file_id bigint        not null,
    nft_tx_hash    varchar(100)  not null,
    mint_address   varchar(100)  not null,
    token_id ...
    primary key pk_source_file_mint_info(id),
    constraint fk_source_file_mint_info_source_file_id foreign key (source_file_id) references source_file(id)
)

create table source_file_upload_history (
    id             bigint        not null auto_increment,
    source_file_id bigint        not null,
    file_name      varchar(200)  not null,
    wallet_id      bigint        not null,
    create_at      bigint        not null,
    update_at      bigint        not null,
    primary key pk_source_file_upload_history(id),
    constraint fk_source_file_upload_history_wallet_id foreign key (wallet_id) references wallet(id)
);

create table deal_file (
    id             bigint        not null auto_increment,
    car_file_name  varchar(200)  not null,
    payload_cid    varchar(200)  not null,
    piece_cid      varchar(200)  not null,
    car_file_size  bigint        not null,
    car_file_path  varchar(1000) not null,
    duration       int           not null,
    task_uuid      varchar(100)  not null,
    max_price      varchar(100)  not null,
    status         varchar(100)  not null,
    create_at      bigint        not null,
    update_at      bigint        not null,
    primary key pk_deal_file(id),
);

create table source_file_deal_file_map (
    id             bigint        not null auto_increment,
    source_file_id bigint        not null,
    deal_file_id   bigint        not null,
    create_at      bigint        not null,
    primary key pk_source_file_deal_file_map(id),
    constraint fk_source_file_deal_file_map_source_file_id foreign key (source_file_id) references source_file(id),
    constraint fk_source_file_deal_file_map_deal_file_id foreign key (deal_file_id) references deal_file(id)
);

create table offline_deal (
    id               bigint        not null auto_increment,
    deal_file_id     bigint        not null,
    deal_cid         varchar(100)  not null,
    miner_id         bigint        not null,
    verified         boolean       not null,
    start_epoch      int           not null,
    sender_wallet_id bigint        not null,
    deal_id          bigint,
    create_at        bigint        not null,
    update_at        bigint        not null,
    primary key pk_offline_deal(id),
    constraint fk_offline_deal_deal_file_id foreign key (deal_file_id) references deal_file(id),
    constraint fk_offline_deal_miner_id foreign key (miner_id) references miner(id)
);

create table offline_deal_log (
    id               bigint        not null auto_increment,
    offline_deal_id  bigint        not null,
    on_chain_status  varchar(100)  not null,
    on_chain_message text,
    create_at        bigint        not null,
    primary key pk_offline_deal_log(id),
    constraint fk_offline_deal_log_offline_deal_id foreign key (offline_deal_id) references offline_deal(id)
);

create table transaction_source_file (
    id               bigint        not null auto_increment,
    source_file_id   bigint        not null,
    type             int           not null, #--0:pay, 1:refund after expired, 2: refund after unlock
    tx_hash          varchar(100)  not null,
    wallet_id_from   bigint        not null,
    wallet_id_to     bigint        not null,
    primary key pk_transaction_source_file(id),
    constraint fk_transaction_source_file_source_file_id foreign key (source_file_id) references source_file(id),
    constraint fk_transaction_source_file_wallet_id_from foreign key (wallet_id_from) references wallet(id),
    constraint fk_transaction_source_file_wallet_id_to foreign key (wallet_id_to) references wallet(id)
);


create table transaction_offline_deal (
    id               bigint        not null auto_increment,
    offline_deal_id  bigint        not null,
    type             int           not null, #--0:unlock
    tx_hash          varchar(100)  not null,
    wallet_id_from   bigint        not null,
    wallet_id_to     bigint        not null,
    primary key pk_transaction_offline_deal(id),
    constraint fk_transaction_offline_deal_offline_deal_id foreign key (offline_deal_id) references offline_deal(id),
    constraint fk_transaction_offline_deal_wallet_id_from foreign key (wallet_id_from) references wallet(id),
    constraint fk_transaction_offline_deal_wallet_id_to foreign key (wallet_id_to) references wallet(id)
)



