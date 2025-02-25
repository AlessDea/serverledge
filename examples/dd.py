import subprocess

tmp = '/tmp'

"""
dd - convert and copy a file
man : http://man7.org/linux/man-pages/man1/dd.1.html

Options 
 - bs=BYTES
    read and write up to BYTES bytes at a time (default: 512);
    overrides ibs and obs
 - if=FILE
    read from FILE instead of stdin
 - of=FILE
    write to FILE instead of stdout
 - count=N
    copy only N input blocks
"""


# def handler(params, context):   
#    bs = 'bs=512'
#    count = 'count='+params['count']
   
#    out_fd = open(tmp + '/io_write_logs', 'w')
#    dd = subprocess.Popen(['dd', 'if=/dev/zero', 'of=/tmp/out', bs, count], stderr=out_fd)
#    dd.communicate()
   
#    subprocess.check_output(['ls', '-alh', tmp])
 
#    with open(tmp + '/io_write_logs') as logs:
#        result = str(logs.readlines()[2]).replace('\n', '')
#        print(result)
#        return result

#if __name__ == "__main__":
#   handler({"bs":1, "count":2}, "ctx")

import subprocess

TMP_DIR = '/tmp'

def handler(params, context):   
    bs = f'bs={params['bs']}'
    
    try:
        count_value = int(params.get('count', 1))  # Default a 1
        if count_value <= 0:
            raise ValueError("count's value must be grather than zero.")
    except ValueError:
        return "Errore: 'count' must be a positive number."

    count = f'count={count_value}'
    log_file_path = f"{TMP_DIR}/io_write_logs"

    # Usa il context manager per gestire il file di log
    with open(log_file_path, 'w') as out_fd:
        result = subprocess.run(['dd', 'if=/dev/zero', 'of=/tmp/out', bs, count], stderr=out_fd, text=True)

    # Controlla la directory
    ls_output = subprocess.run(['ls', '-alh', TMP_DIR], capture_output=True, text=True)
    print(ls_output.stdout)  # Debugging

    # Legge il file di log in modo sicuro
    with open(log_file_path) as logs:
        lines = logs.readlines()
        return lines
      #   if len(lines) >= 3:
      #       result = lines[2].strip()
      #   else:
      #       result = "Errore: dd output is to short."

    print(result)
    return result
