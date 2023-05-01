# typed: true

# DO NOT EDIT MANUALLY
# This is an autogenerated file for types exported from the `standard-custom` gem.
# Please instead update this file by running `bin/tapioca gem standard-custom`.

module RuboCop; end

# Check for uses of braces around single line blocks, but allows either
# braces or do/end for multi-line blocks.
#
# @example
#   # bad - single line block
#   items.each do |item| item / 5 end
#
#   # good - single line block
#   items.each { |item| item / 5 }
#
# source://standard-custom//lib/standard/cop/block_single_line_braces.rb#11
module RuboCop::Cop; end

# source://standard-custom//lib/standard/cop/block_single_line_braces.rb#12
module RuboCop::Cop::Standard; end

# source://standard-custom//lib/standard/cop/block_single_line_braces.rb#13
class RuboCop::Cop::Standard::BlockSingleLineBraces < ::RuboCop::Cop::Base
  extend ::RuboCop::Cop::AutoCorrector

  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#31
  def on_block(node); end

  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#16
  def on_send(node); end

  private

  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#69
  def autocorrect(corrector, node); end

  # @return [Boolean]
  #
  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#75
  def correction_would_break_code?(node); end

  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#43
  def get_blocks(node, &block); end

  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#65
  def message(node); end

  # @return [Boolean]
  #
  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#61
  def proper_block_style?(node); end

  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#81
  def replace_do_end_with_braces(corrector, loc); end

  # @return [Boolean]
  #
  # source://standard-custom//lib/standard/cop/block_single_line_braces.rb#91
  def whitespace_after?(range, length = T.unsafe(nil)); end
end