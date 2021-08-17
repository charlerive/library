import numpy as np
from scipy.optimize import minimize
import sys
import getopt


# ============  TotalVariance Functions ==================================================
def total_variance(k, a, b, rho, eta, c):
    return a + b * (rho * (k - eta) + np.sqrt((k - eta) * (k - eta) + c * c))


def total_variance_pk(params, k):
    a, b, rho, eta, c = params
    return total_variance(k, a, b, rho, eta, c)


# Right constraints
def constraint1(params, k):
    a, b, rho, eta, sig = params
    return ((4 - a + b * eta * (rho + 1)) * (a - b * eta * (rho + 1))) - (b * b * (rho + 1) * (rho + 1))


def constraint2(params, k):
    a, b, rho, eta, sig = params
    return 4 - (b * b * (rho + 1) * (rho + 1))


# Left constraints
def constraint3(params, k):
    a, b, rho, eta, sig = params
    return ((4 - a + b * eta * (rho - 1)) * (a - b * eta * (rho - 1))) - (b * b * (rho - 1) * (rho - 1))


def constraint4(params, k):
    a, b, rho, eta, sig = params
    return 4 - (b * b * (rho - 1) * (rho - 1))


def constraint5(params, k):
    a, b, rho, eta, sig = params
    min_value = 0
    for i, v in enumerate(k):
        cur = (v - eta) ** 2 + sig ** 2
        if min_value > cur:
            min_value = cur
    return min_value


# Objective function to optimize
def least_squares(params, k, tot_implied_variance):
    residual = total_variance_pk(params, k) - tot_implied_variance
    return np.linalg.norm(residual, 2)


k_list = []
v_list = []


def receive_opts():
    global k_list, v_list
    argv = sys.argv[1:]
    try:
        opts, args = getopt.getopt(argv, "hk:v:", ["k_list=", "v_list="])
    except getopt.GetoptError:
        print('main.py -k <k_list> -v <v_list>')
        sys.exit(2)
    for opt, arg in opts:
        if opt == '-h':
            print('main.py -k <k_list> -v <v_list>')
            sys.exit()
        elif opt in ("-k", "--k_list"):
            k_list = eval(arg)
        elif opt in ("-v", "--v_list"):
            v_list = eval(arg)
    if len(k_list) == 0 or len(v_list) == 0:
        print('main.py -k <k_list> -v <v_list>')
        sys.exit(2)
    if len(k_list) != len(v_list):
        print('len(k_list) must eq len(v_list)')
        sys.exit(2)


def minimize_slsqp():
    mkt_k = np.transpose(np.array(k_list))
    mkt_tot_variance = np.transpose(np.array(v_list))

    # ===========  TotalVariance’s Parameters Boundaries ======================================
    a_low = 0.000001
    a_high = np.max(mkt_tot_variance)
    b_low = 0.001
    b_high = 1
    rho_low = -0.999999
    rho_high = 0.999999
    eta_low = 2 * np.min(mkt_k)
    eta_high = 2 * np.max(mkt_k)
    c_low = 0.001
    c_high = 2
    bounds = ((a_low, a_high), (b_low, b_high), (rho_low, rho_high), (eta_low, eta_high), (c_low, c_high))

    # ===========  TotalVariance’s Parameters Initial Guess =====================================
    a_init = np.min(mkt_tot_variance) / 2
    b_init = 0.1
    rho_init = -0.5
    eta_init = 0.1
    c_init = 0.1
    params_init = np.array([a_init,b_init,rho_init,eta_init,c_init])

    # Constraint Function
    cons1 = {'type': 'ineq', 'fun': lambda x: constraint1(x, mkt_k)}
    cons2 = {'type': 'ineq', 'fun': lambda x: constraint2(x, mkt_k)}
    cons3 = {'type': 'ineq', 'fun': lambda x: constraint3(x, mkt_k)}
    cons4 = {'type': 'ineq', 'fun': lambda x: constraint4(x, mkt_k)}
    cons5 = {'type': 'ineq', 'fun': lambda x: constraint5(x, mkt_k)}

    result = minimize(lambda x: least_squares(x, mkt_k, mkt_tot_variance),
                      params_init, method='SLSQP',
                      bounds=bounds,
                      constraints=[cons1, cons2, cons3, cons4, cons5],
                      options={'ftol': 1e-9, 'disp': False})
    a, b, rho, eta, c = result.x
    print(a, ",", b, ",", c, ",", rho, ",", eta)


# python3 minimize_slsqp.py -k '[-0.1524,-0.0879,-0.0273,0.0299,0.0839,0.1352,0.2530]' -v '[0.01018,0.00820,0.00720,0.00597,0.00663,0.00568,0.01289]'
if __name__ == "__main__":
    receive_opts()
    minimize_slsqp()
