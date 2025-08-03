# Requirements Pilot Usage Guide

## 🚀 Overview

The `/requirements-pilot` command provides a complete requirements-driven development workflow that automatically executes a chain of specialized agents to implement features from description to tested code.

## 📝 Usage Syntax

### In GitHub Issues:
```
@claude /requirements-pilot <feature_description>
```

**Example:**
```
@claude /requirements-pilot 添加用户登录功能，支持邮箱和手机号登录，包含记住密码选项
```

## 🔄 Workflow Process

### Phase 1: Requirements Confirmation (Interactive)
1. **Quality Assessment**: Analyzes your feature description using a 100-point scoring system
2. **Clarification Questions**: Asks targeted questions to clarify unclear aspects
3. **Interactive Loop**: Continues until requirements reach ≥90 points quality score
4. **User Approval**: Presents final requirements summary and requests approval to proceed

### Phase 2: Automated Agent Chain (After Approval)
Once you approve (comment `@claude proceed`), the following agents execute automatically:

1. **requirements-generate**: Creates detailed technical specifications
2. **requirements-code**: Implements the functionality based on specs
3. **requirements-review**: Evaluates code quality (0-100% scoring)
4. **Quality Gate**: If ≥90% → testing, if <90% → code improvement (max 3 iterations)
5. **requirements-testing**: Creates comprehensive test suite

## 📊 Quality Scoring System

### Requirements Assessment (100 points):
- **Functional Clarity (30 points)**: Clear input/output specs, user interactions, success criteria
- **Technical Specificity (25 points)**: Integration points, technology constraints, performance requirements
- **Implementation Completeness (25 points)**: Edge cases, error handling, data validation
- **Business Context (20 points)**: User value proposition, priority definition

### Code Quality Assessment (100%):
- **Functional Correctness (40%)**: Does the code solve the specified problem?
- **Integration Quality (30%)**: Does it integrate seamlessly with existing systems?
- **Maintainability (20%)**: Is the code easy to understand and modify?
- **Performance Adequacy (10%)**: Does it perform reasonably for the use case?

## 📁 Generated Artifacts

Each workflow creates a structured directory: `.claude/specs/{feature-name}/`

**Files Generated:**
- `requirements-confirm.md`: Confirmed requirements with quality assessment
- `technical-spec.md`: Implementation-ready technical specifications
- `code-review.md`: Quality assessment and feedback
- `test-suite.md`: Test implementation summary
- `workflow-summary.md`: Complete workflow execution summary

## ⚡ Example Workflow

### Step 1: Initial Request
```
@claude /requirements-pilot 实现文章评论系统，支持多级回复和点赞功能
```

### Step 2: Requirements Clarification
Claude will ask clarifying questions like:
- "需要支持匿名评论吗？"
- "点赞数据如何存储和展示？"
- "多级回复的最大深度是多少？"

### Step 3: Quality Gate Reached
```
Requirements Quality Assessment: 92/100 points
✅ Functional Clarity: 28/30
✅ Technical Specificity: 24/25  
✅ Implementation Completeness: 23/25
✅ Business Context: 17/20

Requirements are now clear (92 points). Do you want to proceed with implementation?
```

### Step 4: User Approval
```
@claude proceed
```

### Step 5: Automated Implementation
- Technical specifications generated
- Code implemented automatically
- Quality review performed
- Tests created
- Final summary provided

## 🛡️ Quality Gates

### Requirements Gate
- **Threshold**: ≥90 points
- **Action**: Continue to implementation phase
- **Failure**: Additional clarification questions

### Code Quality Gate  
- **Threshold**: ≥90% quality score
- **Success**: Proceed to testing
- **Failure**: Return to implementation with specific feedback (max 3 iterations)

## 🔧 Configuration Requirements

### GitHub Secrets Required:
```yaml
ANTHROPIC_API_KEY: Your Anthropic API key
APP_ID: GitHub App ID (for enhanced permissions)
APP_PRIVATE_KEY: GitHub App private key
```

### GitHub Permissions:
- `contents: write` - For code modifications
- `issues: write` - For commenting on issues  
- `pull-requests: write` - For creating PRs

## 📈 Workflow Benefits

- **High Quality**: 90% quality gates ensure production-ready code
- **Complete Automation**: From requirements to tested code
- **Iterative Improvement**: Quality loops refine implementation
- **Full Traceability**: Complete audit trail of decisions and implementations
- **User Control**: Explicit approval gates prevent unwanted execution

## 🎯 Best Practices

### For Better Requirements:
1. **Be Specific**: Include expected behavior, constraints, and success criteria
2. **Provide Context**: Mention integration points and existing systems
3. **Define Scope**: Clearly state what's included and excluded
4. **Include Examples**: Provide concrete use cases or scenarios

### For Optimal Results:
1. **Engage Actively**: Answer clarification questions thoroughly
2. **Review Summaries**: Carefully review requirements before approving
3. **Trust the Process**: Allow quality gates to improve implementation
4. **Monitor Progress**: Track workflow execution through GitHub Actions

## 🔍 Troubleshooting

### Common Issues:
- **Low Quality Score**: Provide more detailed feature descriptions
- **Failed Approval**: Check that requirements confirmation file exists
- **Agent Failures**: Review GitHub Actions logs for specific errors
- **Permission Errors**: Verify GitHub App and API key configuration

### Recovery Actions:
- **Re-run Requirements**: Use `/requirements-pilot` again with more details
- **Manual Approval**: Comment `@claude proceed` if approval step was missed
- **Check Artifacts**: Review generated files in `.claude/specs/` directory