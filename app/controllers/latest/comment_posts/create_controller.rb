# typed: true
# frozen_string_literal: true

class Latest::CommentPosts::CreateController < Latest::ApplicationController
  def call
    form = Forms::CommentPost.new(
      comment: params[:comment]
    )
    form.profile = current_profile!

    if form.invalid?
      return render(
        json: Resources::Latest::ActiveModelErrors.new(form.errors),
        status: :unprocessable_entity
      )
    end

    result = ActiveRecord::Base.transaction do
      Services::CreateCommentPost.new(form:).call
    end

    render(
      json: Resources::Latest::Post.new(result.post),
      status: :created
    )
  end
end
