TEST_DIR_PARALLEL=(
  "mounting"
  "readonly"
  "operations"
  "explicit_dir"
  "log_rotation"
  "gzip"
  "write_large_files"
  "implicit_dir"
  "local_file"
  "read_large_files"
  "rename_dir_limit"
  "read_cache"
)
for test_dir_np in "${TEST_DIR_PARALLEL[@]}"
do
   test_path_non_parallel="./tools/integration_tests/$test_dir_np"
   # Executing integration tests
   GODEBUG=asyncpreemptoff=1 go test $test_path_non_parallel -p 1 --integrationTest -v --testbucket=tulsishah-tpc-bucket -timeout 120m --testInstalledPackage
done