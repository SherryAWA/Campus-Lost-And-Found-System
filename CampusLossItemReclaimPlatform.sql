drop trigger Studemt_BEFORE_DELETE;

drop procedure if exists CountLostItems;

drop table if exists 丢失物品查询;

drop index Index_ItemID on ClaimForm;

create table Admin
(
   ID                   varchar(15) not null,
   Password             varchar(20) not null,
   primary key (ID)
);

create table ClaimForm
(
   ClaimFormID          varchar(15) not null,
   ItemID               varchar(15) not null,
   ApplyTime            datetime not null,
   Status               varchar(10) not null,
   ID                   varchar(15) not null,
   primary key (ClaimFormID)
);

create index Index_ItemID on ClaimForm
(
   ItemID
);

create table Complaint
(
   ComplaintID          varchar(15) not null,
   ID                   varchar(15) not null,
   Adm_ID               varchar(15),
   ClaimFormID          varchar(15) not null,
   UserID               varchar(15) not null,
   Category             varchar(15) not null,
   Reason               varchar(100) not null,
   Time                 datetime not null,
   Advice               longtext,
   Time2                datetime,
   primary key (ComplaintID)
);

create table DiscreditedList
(
   UserID               varchar(15) not null,
   ID                   varchar(15) not null,
   ColdTime             datetime not null,
   DecoldTime           datetime not null,
   primary key (UserID)
);

create table FoundItem
(
   ItemID               varchar(15) not null,
   ID                   varchar(15) not null,
   ClaimFormID          varchar(15),
   Category             varchar(15),
   ItemName             varchar(15) not null,
   Description          varchar(100),
   Location             varchar(15),
   Time                 datetime,
   primary key (ItemID)
);

create table LossItem
(
   ItemID               varchar(15) not null,
   ID                   varchar(15) not null,
   Category             varchar(15),
   ItemName             varchar(15) not null,
   Description          varchar(100),
   Location             varchar(15),
   Time                 datetime,
   primary key (ItemID)
);

create table Student
(
   ID                   varchar(15) not null,
   Name                 varchar(10) not null,
   Gender               varchar(5) not null,
   UserType             varchar(15) not null,
   Status               varchar(15) not null,
   Telephone            varchar(13) not null,
   CardNumber           varchar(18) not null,
   primary key (ID)
);

create table User
(
   ID                   varchar(15) not null,
   Name                 varchar(20) not null,
   Status               bool not null,
   primary key (ID)
);

alter table User comment '账号：学号
密码：身份证号后六位
账号状态：0表示未被冻结 1表示已被冻结(无法登录)';

create VIEW  丢失物品查询 
as 
select Student.ID,Student.Name,LossItem.ItemName from Student 
join User 
on Student.ID=User.ID 
join LossItem 
on LossItem.ID=User.ID;

alter table ClaimForm add constraint FK_Claim foreign key (ID)
      references User (ID) on delete restrict on update restrict;

alter table Complaint add constraint FK_Complain foreign key (ID)
      references User (ID) on delete restrict on update restrict;

alter table Complaint add constraint FK_handle foreign key (Adm_ID)
      references Admin (ID) on delete restrict on update restrict;

alter table DiscreditedList add constraint FK_Update foreign key (ID)
      references Admin (ID) on delete restrict on update restrict;

alter table FoundItem add constraint FK_include foreign key (ClaimFormID)
      references ClaimForm (ClaimFormID) on delete restrict on update restrict;

alter table FoundItem add constraint FK_发布 foreign key (ID)
      references User (ID) on delete restrict on update restrict;

alter table LossItem add constraint FK_PostLossItems foreign key (ID)
      references User (ID) on delete restrict on update restrict;

alter table User add constraint "FK_Student-User" foreign key (ID)
      references Student (ID) on delete restrict on update restrict;


create procedure %CountLostItemsCountLostItems(IN student_ID VARCHAR(15), OUT lostItemCount INT)
BEGIN
    SELECT COUNT(*) INTO lostItemCount
    FROM LostItem
    WHERE ID = student_ID;
END 
DELIMITER ;


CREATE TRIGGER Student_BEFORE_DELETE 
BEFORE DELETE ON Student
FOR EACH ROW
BEGIN
    DELETE FROM User 
    WHERE ID = OLD.ID;
END;

