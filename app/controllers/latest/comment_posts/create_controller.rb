# typed: true
# frozen_string_literal: true

class Latest::CommentPosts::CreateController < Latest::ApplicationController
  def call
    form = Forms::CommentPost.new(
      comment: params[:comment]
    )
    form.profile = current_profile!

    if form.invalid?
      resource_errors = Latest::Entities::FormError.build_from_errors(errors: form.errors)

      return render(
        json: Latest::Resources::ResponseError.new(resource_errors),
        status: :unprocessable_entity
      )
    end

    result = ActiveRecord::Base.transaction do
      Services::CreateCommentPost.new(form:).call
    end

    render(
      json: Latest::Resources::Post.new(Latest::Entities::Post.new(post: result.post, viewer: current_profile!)),
      status: :created
    )
  end
end
