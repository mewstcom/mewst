# typed: true
# frozen_string_literal: true

class Api::Internal::Posts::CreateController < ApplicationController
  include Authenticatable

  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    post_creator = T.must(current_profile).new_post(
      content: params[:content]
    )

    if post_creator.invalid?
      return render(json: {errors: post_creator.errors.full_messages}, status: :unprocessable_entity)
    end

    post_creator.call

    @post = T.must(post_creator.post)

    render(status: :created)
  end
end
