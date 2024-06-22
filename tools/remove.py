import getopt
import sys


def read(path):
    origin_data = open(path, 'r', encoding='utf-8').read().splitlines()
    return list(filter(None, origin_data))


def main(argv):
    remove_file = ""
    source_file = ""
    output_file = ""
    try:
        opts, args = getopt.getopt(argv, "r:s:o:",
                                   ["remove=", "source=", "output="])
    except getopt.GetoptError:
        print("exit")
        sys.exit(2)
    for opt, arg in opts:
        if opt in ("-r", "--remove"):
            remove_file = arg
        elif opt in ("-s", "--source"):
            source_file = arg
        elif opt in ("-o", "--output"):
            output_file = arg

    source_data = read(source_file)
    remove_data = read(remove_file)
    for i in remove_data:
        try:
            source_data.remove(i)
        except ValueError:
            print("%s not in %s" % (i, source_file))
    output_data = '\n'.join(source_data)
    file_handle = open(output_file, mode='w')
    file_handle.writelines(output_data)
    file_handle.close()


if __name__ == "__main__":
    main(sys.argv[1:])
