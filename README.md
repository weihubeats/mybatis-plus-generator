# mybatis-plus-generator
mybatis plus java code generator,but is use go(mybatis plus 代码自动生成器，但是是用golang实现，简单便捷，开箱即用，无需安装任何其他依赖)

## 使用

## 源码方式运行

直接运行`main`方法

访问 http://localhost:8080 进入主界面

![home.png](doc/home.png)

输入sql

比如

```sql
create table "order"
(
    id           bigserial,
    uid          bigint  not null,
    order_id     bigint  not null,
    order_status text,
    product_num  integer not null,
    device_id    text,
    device       text,
    version      text,
    add_time     timestamp(0) default now(),
    update_time  timestamp(0),
    influhub_uid bigint,
    pay_amount   numeric
);

comment on table influhub_order is '订单表';

comment on column influhub_order.uid is '订单uid';

comment on column influhub_order.order_id is '订单id';

comment on column influhub_order.order_status is '订单状态(ORDER_CREATE（下单）、ORDER_CANCEL（全单取消）、ORDER_PART_CANCEL（部分取消）、ORDER_SIGN（确认收货）、ORDER_REFUND（订单退款）)';

comment on column influhub_order.currency is '币种';

comment on column influhub_order.product_num is '商品数量';

comment on column influhub_order.device_id is '设备id';

comment on column influhub_order.device is '渠道(web、pc)';

comment on column influhub_order.version is '订单版本';

comment on column influhub_order.influhub_uid is 'uid';

comment on column influhub_order.pay_amount is '订单支付金额';
```

注意表名如果是数据库保留管关键字，需要加""

- 效果

![example.png](doc/example.png)


> 目前仅对Postgresql进行了测试,mysql还没有测试

## 安装包运行方式

[releases](https://github.com/weihubeats/mybatis-plus-generator/releases)页面下载符合自己系统的二进制可执行文件

下载完直接双击运行。然后在浏览器访问`http://localhost:8080`



# 为什么要写这个库

市面上的`mybatis plus` 代码生成器有很多，但是都不太满足自己的要求。

在使用`mybatis plus`自己会有一个规范，就是不要将`mybatis plus`的`Warpper`暴露到`DAO`以外的层级，特别是`Service`层

不然造成的后果就是`Warpper`在`Service`满天飞。然后有如下弊端

1. `Service`层充满了数据查询的各种`Warpper`，逻辑不够解耦，导致应该在数据层的逻辑全部暴露在`Service`层
2. `Service`层的数据查询没有封装无法复用

所以我们一般使用`mybatis plus`的标准用法结构是

- infra
  - dao
    - mapper
      - OrderMapper.java
    - impl
      - OrderDAOImpl.java
  OrderDAO.java
  - entity
    - OrderDO.java

具体的代码如下：

- OrderDO

```java
@Data
public class OrderDO {

}
```

- OrderMapper

```java
@Mapper
public interface OrderMapper extends BaseMapper<OrderDO> {
}
```

- OrderDAO

```java
public interface OrderDAO extends IService<OrderDO> {
}
```

- OrderDAOImpl

```java
@Repository
@RequiredArgsConstructor
public class OrderDAOImpl extends ServiceImpl<OrderMapper, OrderDO> implements OrderDAO {

    private final InfluhubOrderMapper influhub_orderMapper;
}
```

