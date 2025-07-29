# typed: strict
# frozen_string_literal: true

class Cards::PostCard::BodyComponent < ApplicationComponent
  sig { params(post: PostRecord, class_name: String).void }
  def initialize(post:, class_name: "")
    @post = post
    @class_name = class_name
  end

  sig { returns(PostRecord) }
  attr_reader :post
  private :post

  sig { returns(String) }
  attr_reader :class_name
  private :class_name

  sig { returns(ProfileRecord) }
  private def profile
    post.profile_record.not_nil!
  end
end
