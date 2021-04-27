control_group "cg_1"{
}
control_group "cg_1_1"{
    parent = "control_group.cg_1"
}
control_group "cg_1_2"{
    parent = "control_group.cg_1"
}
control_group "cg_1_1_1"{
    parent = "control_group.cg_1_1"
}
control_group "cg_1_1_2"{
    parent = "control_group.cg_1_1"
    documentation="foo"
}
control "c1"{
    description = "control 1"
    sql = "query.q1"
    parent = "control_group.cg_1_1_1"
}
control "c2"{
    description = "control 2"
    sql = "select 'control 2' as control, 'pass' as result"
    parent = "control_group.cg_1_1_2"
}
control "c3"{
    description = "control 3"
    sql = "select 'control 3' as control, 'pass' as result"
    parent = "control_group.cg_1_1"
}
control "c4"{
    description = "control 4"
    sql = "select 'control 4' as control, 'pass' as result"
    parent = "control_group.cg_1_1_2"
}
control "c5"{
    description = "control 5"
    sql = "select 'control 5' as control, 'pass' as result"
    parent = "control_group.cg_1_1_2"
}
control "c6"{
    description = "control 6"
    sql = "select 'control 6' as control, 'FAIL' as result"
    // no parent - under mod
}