const { exec } = require("child_process");

// 检查是否提供了新的版本作为参数
if (process.argv.length < 3) {
  console.error("请提供新的版本作为参数，例如：node scripts/prepare.js v1.0.0");
  process.exit(1);
}

// 从参数中获取新的版本
const newVersion = process.argv[2];

exec(
  `cd config/manager && \
  ../../bin/kustomize edit set image controller=ghcr.io/kudeploy/operator:${newVersion}`,
  (error, stdout, stderr) => {
    if (error) {
      console.error(`执行命令时出错: ${error.message}`);
      return;
    }

    if (stderr) {
      console.error(`stderr: ${stderr}`);
      return;
    }
  }
);
