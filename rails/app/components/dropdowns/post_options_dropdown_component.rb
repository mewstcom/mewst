# typed: strict
# frozen_string_literal: true

class Dropdowns::PostOptionsDropdownComponent < ApplicationComponent
  sig { params(post: PostRecord).void }
  def initialize(post:)
    @post = post
  end

  sig { returns(PostRecord) }
  attr_reader :post
  private :post
end
