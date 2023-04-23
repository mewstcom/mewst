# typed: strict
# frozen_string_literal: true

class Commands::CreateRepost < Commands::Base
  sig { params(form: Forms::Repost).void }
  def initialize(form:)
    @form = form
  end

  sig { returns(Post) }
  def execute
    post = profile.create_repost(target_post: form.post)
    post
  end
end
